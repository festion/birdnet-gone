#!/usr/bin/env python3
"""VicoHome Bridge — polls VicoHome cloud API and publishes bird detections to MQTT."""

import os
import sys
import json
import time
import base64
import signal
import logging
from pathlib import Path
from datetime import datetime, timedelta, timezone

import requests as http_requests
import paho.mqtt.client as mqtt

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

def load_env(path=".env"):
    """Load .env file into os.environ if it exists."""
    env_path = Path(path)
    if env_path.exists():
        for line in env_path.read_text().splitlines():
            line = line.strip()
            if line and not line.startswith("#") and "=" in line:
                key, _, value = line.partition("=")
                value = value.strip()
                if len(value) >= 2 and value[0] == value[-1] and value[0] in ('"', "'"):
                    value = value[1:-1]
                os.environ.setdefault(key.strip(), value)

load_env()

VICOHOME_EMAIL = os.environ.get("VICOHOME_EMAIL", "")
VICOHOME_PASSWORD = os.environ.get("VICOHOME_PASSWORD", "")
MQTT_BROKER = os.environ.get("MQTT_BROKER", "192.168.1.148")
MQTT_PORT = int(os.environ.get("MQTT_PORT", "1883"))
MQTT_USERNAME = os.environ.get("MQTT_USERNAME", "birdnet")
MQTT_PASSWORD = os.environ.get("MQTT_PASSWORD", "")
POLL_INTERVAL = int(os.environ.get("POLL_INTERVAL", "60"))

VICOHOME_API = "https://api-us.vicohome.io"
STATE_FILE = Path(__file__).parent / ".last_event"
DETECTIONS_FILE = Path(__file__).parent / "latest_detections.json"
TOKEN_FILE = Path(__file__).parent / ".token_cache"

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)
log = logging.getLogger("vicohome-bridge")


# ---------------------------------------------------------------------------
# VicoHome API Client
# ---------------------------------------------------------------------------

class VicoHomeClient:
    """Client for the VicoHome cloud API."""

    AUTH_ERRORS = {-1024, -1025, -1026, -1027}

    def __init__(self, email, password):
        self.email = email
        self.password = password
        self.token = None
        self._load_cached_token()

    def _load_cached_token(self):
        """Load token from disk cache if still valid."""
        if TOKEN_FILE.exists():
            try:
                data = json.loads(TOKEN_FILE.read_text())
                expires = datetime.fromisoformat(data["expires"])
                if datetime.now(timezone.utc) < expires:
                    self.token = data["token"]
                    log.info("Loaded cached auth token")
                    return
            except (json.JSONDecodeError, KeyError, ValueError):
                pass
        self.token = None

    def _save_token(self):
        """Cache token to disk with 23-hour expiry."""
        data = {
            "token": self.token,
            "expires": (datetime.now(timezone.utc) + timedelta(hours=23)).isoformat(),
        }
        TOKEN_FILE.write_text(json.dumps(data))

    def authenticate(self):
        """Authenticate with VicoHome API and cache the token."""
        log.info("Authenticating with VicoHome API...")
        resp = http_requests.post(
            f"{VICOHOME_API}/account/login",
            json={"email": self.email, "password": self.password, "loginType": 0},
            headers={"Content-Type": "application/json", "Accept": "application/json"},
            timeout=15,
        )
        resp.raise_for_status()
        body = resp.json()
        result = body.get("result", -1)
        if result != 0:
            raise RuntimeError(f"VicoHome login failed: {body.get('msg', 'unknown')} (code {result})")
        self.token = body["data"]["token"]["token"]
        self._save_token()
        log.info("Authentication successful")

    def _request(self, method, path, **kwargs):
        """Make an authenticated API request with auto-retry on token expiry."""
        if not self.token:
            self.authenticate()
        headers = kwargs.pop("headers", {})
        headers["Authorization"] = self.token
        resp = http_requests.request(method, f"{VICOHOME_API}{path}", headers=headers, timeout=15, **kwargs)
        resp.raise_for_status()
        body = resp.json()
        code = body.get("code", body.get("result", 0))
        if code in self.AUTH_ERRORS:
            log.warning("Token expired, re-authenticating...")
            self.authenticate()
            headers["Authorization"] = self.token
            resp = http_requests.request(method, f"{VICOHOME_API}{path}", headers=headers, timeout=15, **kwargs)
            resp.raise_for_status()
            body = resp.json()
        return body

    def get_events(self, start_time=None):
        """Fetch recent bird detection events.

        Args:
            start_time: Unix timestamp. Only events after this time are returned.
                        Defaults to 24 hours ago.

        Returns:
            List of normalized event dicts with keys: traceId, timestamp, birdName,
            birdLatin, birdConfidence, keyShotUrl, imageUrl, videoUrl, deviceName,
            serialNumber
        """
        now = int(datetime.now(timezone.utc).timestamp())
        if start_time is None:
            start_time = now - 86400  # 24 hours ago

        body = self._request("POST", "/library/newselectlibrary", json={
            "startTimestamp": str(start_time),
            "endTimestamp": str(now),
            "language": "en",
            "countryNo": "US",
        })

        events = []
        raw_list = body.get("data", {}).get("list", [])
        if not isinstance(raw_list, list):
            return events

        for raw in raw_list:
            # Extract bird info from subcategoryInfoList
            bird_name = ""
            bird_latin = ""
            bird_confidence = 0.0
            for sub in raw.get("subcategoryInfoList", []):
                if sub.get("objectType") == "bird":
                    bird_name = sub.get("objectName", "")
                    bird_latin = sub.get("birdStdName", "")
                    bird_confidence = sub.get("confidence", 0.0)
                    break

            # Get keyshot URL if available
            keyshot_url = ""
            keyshots = raw.get("keyshots", [])
            if keyshots and isinstance(keyshots, list):
                keyshot_url = keyshots[0].get("imageUrl", "")

            events.append({
                "traceId": raw.get("traceId", ""),
                "timestamp": raw.get("timestamp", 0),
                "birdName": bird_name,
                "birdLatin": bird_latin,
                "birdConfidence": bird_confidence,
                "keyShotUrl": keyshot_url,
                "imageUrl": raw.get("imageUrl", ""),
                "videoUrl": raw.get("videoUrl", ""),
                "deviceName": raw.get("deviceName", ""),
                "serialNumber": raw.get("serialNumber", ""),
            })

        return events


# ---------------------------------------------------------------------------
# MQTT Publisher with Home Assistant Discovery
# ---------------------------------------------------------------------------

NODE_ID = "vicohome_bridge"
DEVICE_INFO = {
    "identifiers": ["vicohome_g02_bridge"],
    "name": "VicoHome Bird Feeder",
    "manufacturer": "HeaPets",
    "model": "G02",
    "sw_version": "1.0.0",
}
BASE_TOPIC = "vicohome"
DISCOVERY_PREFIX = "homeassistant"

def create_mqtt_client():
    """Create and connect an MQTT client with LWT."""
    client = mqtt.Client(mqtt.CallbackAPIVersion.VERSION2, client_id="vicohome-bridge")
    if MQTT_USERNAME:
        client.username_pw_set(MQTT_USERNAME, MQTT_PASSWORD)
    client.will_set(f"{BASE_TOPIC}/status", "offline", qos=1, retain=True)
    client.connect(MQTT_BROKER, MQTT_PORT, keepalive=120)
    client.loop_start()
    return client

def publish_discovery(client):
    """Publish HA MQTT auto-discovery configs for all entities."""

    # Binary sensor: bridge online status
    client.publish(
        f"{DISCOVERY_PREFIX}/binary_sensor/{NODE_ID}/status/config",
        json.dumps({
            "name": "VicoHome Bridge",
            "unique_id": f"{NODE_ID}_status",
            "device": DEVICE_INFO,
            "state_topic": f"{BASE_TOPIC}/status",
            "payload_on": "online",
            "payload_off": "offline",
            "device_class": "connectivity",
        }),
        qos=1, retain=True,
    )

    # Sensor: latest bird species
    client.publish(
        f"{DISCOVERY_PREFIX}/sensor/{NODE_ID}/latest_species/config",
        json.dumps({
            "name": "Latest Species",
            "unique_id": f"{NODE_ID}_latest_species",
            "device": DEVICE_INFO,
            "state_topic": f"{BASE_TOPIC}/latest",
            "value_template": "{{ value_json.bird_name }}",
            "json_attributes_topic": f"{BASE_TOPIC}/latest",
            "icon": "mdi:bird",
            "availability_topic": f"{BASE_TOPIC}/status",
            "payload_available": "online",
            "payload_not_available": "offline",
        }),
        qos=1, retain=True,
    )

    # Sensor: confidence
    client.publish(
        f"{DISCOVERY_PREFIX}/sensor/{NODE_ID}/confidence/config",
        json.dumps({
            "name": "Detection Confidence",
            "unique_id": f"{NODE_ID}_confidence",
            "device": DEVICE_INFO,
            "state_topic": f"{BASE_TOPIC}/latest",
            "value_template": "{{ (value_json.confidence * 100) | round(0) if value_json.confidence <= 1 else value_json.confidence | round(0) }}",
            "unit_of_measurement": "%",
            "icon": "mdi:percent",
            "availability_topic": f"{BASE_TOPIC}/status",
            "payload_available": "online",
            "payload_not_available": "offline",
        }),
        qos=1, retain=True,
    )

    # Sensor: detection count today
    client.publish(
        f"{DISCOVERY_PREFIX}/sensor/{NODE_ID}/detections_today/config",
        json.dumps({
            "name": "Detections Today",
            "unique_id": f"{NODE_ID}_detections_today",
            "device": DEVICE_INFO,
            "state_topic": f"{BASE_TOPIC}/stats",
            "value_template": "{{ value_json.detections_today }}",
            "icon": "mdi:counter",
            "availability_topic": f"{BASE_TOPIC}/status",
            "payload_available": "online",
            "payload_not_available": "offline",
        }),
        qos=1, retain=True,
    )

    # Camera: latest bird image
    client.publish(
        f"{DISCOVERY_PREFIX}/camera/{NODE_ID}/snapshot/config",
        json.dumps({
            "name": "Bird Snapshot",
            "unique_id": f"{NODE_ID}_snapshot",
            "device": DEVICE_INFO,
            "topic": f"{BASE_TOPIC}/snapshot",
            "image_encoding": "b64",
            "availability_topic": f"{BASE_TOPIC}/status",
            "payload_available": "online",
            "payload_not_available": "offline",
        }),
        qos=1, retain=True,
    )

    log.info("Published HA discovery configs")

def publish_detection(client, event):
    """Publish a bird detection event to MQTT."""
    payload = {
        "bird_name": event["birdName"],
        "scientific_name": event["birdLatin"],
        "confidence": event["birdConfidence"],
        "image_url": event["keyShotUrl"] or event["imageUrl"],
        "video_url": event["videoUrl"],
        "timestamp": event["timestamp"],
        "device_name": event["deviceName"],
        "trace_id": event["traceId"],
        "source": "vicohome_g02",
    }
    client.publish(f"{BASE_TOPIC}/latest", json.dumps(payload), qos=1, retain=True)
    client.publish(f"{BASE_TOPIC}/detection", json.dumps(payload), qos=1, retain=False)
    conf = event["birdConfidence"]
    conf_pct = conf if conf > 1 else conf * 100
    log.info(f"Published: {event['birdName']} ({conf_pct:.0f}%)")
    publish_snapshot(client, payload["image_url"])

def publish_stats(client, detections_today):
    """Publish daily stats."""
    client.publish(
        f"{BASE_TOPIC}/stats",
        json.dumps({"detections_today": detections_today}),
        qos=1, retain=True,
    )

def publish_snapshot(client, image_url):
    """Download bird image and publish as base64 for HA camera entity."""
    if not image_url:
        return
    try:
        resp = http_requests.get(image_url, timeout=10)
        resp.raise_for_status()
        b64_image = base64.b64encode(resp.content).decode("ascii")
        client.publish(f"{BASE_TOPIC}/snapshot", b64_image, qos=1, retain=True)
        log.debug(f"Published snapshot ({len(resp.content)} bytes)")
    except http_requests.exceptions.RequestException as e:
        log.warning(f"Failed to download snapshot: {e}")


def write_detections_file(events):
    """Write recent camera detections to a JSON file for the display app."""
    detections = []
    for e in events:
        if not e.get("birdName"):
            continue
        conf = e["birdConfidence"]
        detections.append({
            "name": e["birdName"],
            "scientific_name": e.get("birdLatin", ""),
            "confidence": conf if conf > 1 else conf * 100,
            "image_url": e.get("keyShotUrl") or e.get("imageUrl", ""),
            "video_url": e.get("videoUrl", ""),
            "timestamp": e.get("timestamp", 0),
            "source": "camera",
        })
    # Keep only latest 10
    detections = detections[:10]
    try:
        DETECTIONS_FILE.write_text(json.dumps(detections, indent=2))
    except Exception as ex:
        log.warning(f"Failed to write detections file: {ex}")


def load_last_seen():
    """Load the last-seen event traceId from disk."""
    if STATE_FILE.exists():
        return STATE_FILE.read_text().strip()
    return None


def save_last_seen(trace_id):
    """Save the last-seen event traceId to disk."""
    STATE_FILE.write_text(trace_id)


running = True

def handle_signal(sig, frame):
    global running
    log.info(f"Received signal {sig}, shutting down...")
    running = False

def main():
    global running

    if not VICOHOME_EMAIL or not VICOHOME_PASSWORD:
        log.error("VICOHOME_EMAIL and VICOHOME_PASSWORD must be set")
        sys.exit(1)

    signal.signal(signal.SIGTERM, handle_signal)
    signal.signal(signal.SIGINT, handle_signal)

    # Initialize clients
    vico = VicoHomeClient(VICOHOME_EMAIL, VICOHOME_PASSWORD)
    mqtt_client = create_mqtt_client()
    time.sleep(1)

    # Publish discovery and online status
    publish_discovery(mqtt_client)
    mqtt_client.publish(f"{BASE_TOPIC}/status", "online", qos=1, retain=True)

    last_seen = load_last_seen()
    detections_today = 0
    today = datetime.now().strftime("%Y-%m-%d")

    log.info(f"Polling every {POLL_INTERVAL}s (last_seen={last_seen})")

    # Write initial detections file on startup
    try:
        start = int((datetime.now(timezone.utc) - timedelta(hours=2)).timestamp())
        initial_events = vico.get_events(start_time=start)
        write_detections_file(initial_events)
        log.info(f"Wrote {len([e for e in initial_events if e.get('birdName')])} camera detections to state file")
    except Exception as e:
        log.warning(f"Failed to write initial detections: {e}")

    while running:
        try:
            # Reset daily counter at midnight
            current_date = datetime.now().strftime("%Y-%m-%d")
            if current_date != today:
                detections_today = 0
                today = current_date

            # Fetch events from last 2 hours to catch anything missed
            start = int((datetime.now(timezone.utc) - timedelta(hours=2)).timestamp())
            events = vico.get_events(start_time=start)

            # Process only events newer than last_seen
            new_events = []
            for e in events:
                if last_seen and e["traceId"] == last_seen:
                    break
                new_events.append(e)

            # Publish in chronological order (oldest first)
            for event in reversed(new_events):
                if event["birdName"]:  # Skip non-bird motion events
                    publish_detection(mqtt_client, event)
                    detections_today += 1

            if new_events:
                last_seen = new_events[0]["traceId"]
                save_last_seen(last_seen)
                publish_stats(mqtt_client, detections_today)
                write_detections_file(events)  # Update state file for display app
                log.info(f"Processed {len(new_events)} new events ({detections_today} today)")

        except http_requests.exceptions.RequestException as e:
            log.warning(f"API request failed: {e}")
        except Exception as e:
            log.error(f"Unexpected error: {e}", exc_info=True)

        # Sleep in small increments for responsive shutdown
        for _ in range(POLL_INTERVAL):
            if not running:
                break
            time.sleep(1)

    # Clean shutdown
    log.info("Shutting down...")
    mqtt_client.publish(f"{BASE_TOPIC}/status", "offline", qos=1, retain=True)
    time.sleep(0.5)
    mqtt_client.loop_stop()
    mqtt_client.disconnect()
    log.info("Goodbye")

if __name__ == "__main__":
    main()
