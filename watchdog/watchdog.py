#!/usr/bin/env python3
"""BirdNET Audio Watchdog — detects BOYA mic desync via silence monitoring."""

import json
import os
import sys
import time
import logging
from datetime import datetime, timezone, timedelta
import paho.mqtt.client as mqtt
import requests

# --- Configuration (from environment) ---
MQTT_BROKER = os.environ.get("MQTT_BROKER", "localhost")
MQTT_PORT = int(os.environ.get("MQTT_PORT", "1883"))
MQTT_USER = os.environ.get("MQTT_USER", "")
MQTT_PASS = os.environ.get("MQTT_PASS", "")
MQTT_TOPIC = os.environ.get("MQTT_TOPIC", "birdnet/soundlevel")

NOTIFY_URL = os.environ.get("NOTIFY_URL", "")

# Bird-relevant frequency bands (1-8 kHz) — keys in the soundlevel payload
BIRD_BANDS = ["1.0_kHz", "1.2_kHz", "1.6_kHz", "2.0_kHz", "2.5_kHz",
              "3.1_kHz", "4.0_kHz", "5.0_kHz", "6.3_kHz", "8.0_kHz"]

# If the mean dB across bird bands stays below this, the mic is likely dead.
# A working mic picks up ambient noise at -80 to -90 dB even in silence.
# A dead/desynced mic reads -100 to -110+ (digital silence / noise floor).
SILENCE_THRESHOLD_DB = -97.0

# How long silence must persist before alerting (seconds)
SILENCE_ALERT_AFTER = 30 * 60  # 30 minutes

# Daytime hours (CST) — only alert during these hours
DAYTIME_START_HOUR = 6
DAYTIME_END_HOUR = 20

# Cooldown: don't re-alert for this long after sending one
ALERT_COOLDOWN = 2 * 60 * 60  # 2 hours

# Use system local timezone


logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S"
)
log = logging.getLogger("birdnet-watchdog")

if not MQTT_PASS:
    log.error("MQTT_PASS environment variable is required.")
    log.error("Copy watchdog/.env.example to watchdog/.env and fill in values.")
    sys.exit(1)
if not NOTIFY_URL:
    log.warning("NOTIFY_URL not set — alerts will be disabled")

# --- State ---
silence_start = None
last_alert_time = 0
last_good_audio_time = time.time()


def is_daytime():
    now = datetime.now().astimezone()
    return DAYTIME_START_HOUR <= now.hour < DAYTIME_END_HOUR


def compute_mean_db(bands_data):
    """Compute average mean dB across bird-frequency bands."""
    values = []
    for band_key in BIRD_BANDS:
        band = bands_data.get(band_key)
        if band and "m" in band:
            values.append(band["m"])
    if not values:
        return None
    return sum(values) / len(values)


def send_alert():
    global last_alert_time
    now = time.time()
    if now - last_alert_time < ALERT_COOLDOWN:
        log.info("Alert suppressed (cooldown active)")
        return

    try:
        payload = {
            "audience": "jeremy",
            "title": "BirdNET Mic Desync",
            "message": "BOYA Magic mic appears desynced — zero audio for 30+ minutes. Manual resync needed.",
            "category": "system",
            "criticality": "high",
            "alexa": False
        }
        resp = requests.post(NOTIFY_URL, json=payload, timeout=10)
        if resp.status_code == 200:
            log.warning("Alert sent: mic desync detected")
            last_alert_time = now
        else:
            log.error(f"Alert failed: HTTP {resp.status_code}")
    except Exception as e:
        log.error(f"Alert send error: {e}")


def on_connect(client, userdata, flags, reason_code, properties):
    if reason_code == 0:
        log.info(f"Connected to MQTT broker at {MQTT_BROKER}")
        client.subscribe(MQTT_TOPIC)
    else:
        log.error(f"MQTT connect failed: {reason_code}")


def on_message(client, userdata, msg):
    global silence_start, last_good_audio_time

    try:
        data = json.loads(msg.payload)
        bands = data.get("b", {})
        mean_db = compute_mean_db(bands)

        if mean_db is None:
            return

        is_silent = mean_db < SILENCE_THRESHOLD_DB
        now = time.time()

        if is_silent:
            if silence_start is None:
                silence_start = now
                log.info(f"Silence detected (mean={mean_db:.1f} dB)")

            silence_duration = now - silence_start

            if silence_duration >= SILENCE_ALERT_AFTER and is_daytime():
                mins = int(silence_duration / 60)
                log.warning(f"Silence for {mins}min (mean={mean_db:.1f} dB) — triggering alert")
                send_alert()
        else:
            if silence_start is not None:
                duration = int((now - silence_start) / 60)
                log.info(f"Audio resumed after {duration}min silence (mean={mean_db:.1f} dB)")
            silence_start = None
            last_good_audio_time = now

    except (json.JSONDecodeError, KeyError) as e:
        log.error(f"Parse error: {e}")


def main():
    log.info("BirdNET Audio Watchdog starting")
    log.info(f"Silence threshold: {SILENCE_THRESHOLD_DB} dB, alert after: {SILENCE_ALERT_AFTER/60:.0f}min")
    log.info(f"Daytime window: {DAYTIME_START_HOUR}:00-{DAYTIME_END_HOUR}:00")

    client = mqtt.Client(mqtt.CallbackAPIVersion.VERSION2, client_id="birdnet-watchdog")
    client.username_pw_set(MQTT_USER, MQTT_PASS)
    client.on_connect = on_connect
    client.on_message = on_message

    client.connect(MQTT_BROKER, MQTT_PORT, keepalive=60)
    client.loop_forever()


if __name__ == "__main__":
    main()
