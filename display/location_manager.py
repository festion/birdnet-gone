#!/usr/bin/env python3
"""
BirdNET Display - Location Manager
Startup service to manage location detection and species cache updates.

This script runs on system startup to:
1. Detect device location (IP geolocation, GPS, or manual config)
2. Update BirdNET-Go configuration with coordinates
3. Fetch location-specific species list
4. Update image cache incrementally (only new species)

Exit Codes:
  0  Success - location updated and cache synchronized
  1  Error - critical failure (permissions, API down, etc.)
  2  No changes - location unchanged, cache up to date
"""

import os
import sys
import logging
from pathlib import Path

# Add utils directory to path
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from utils.geolocation import LocationDetector
from utils.config_manager import BirdNETConfigManager
import subprocess

# --- Configuration ---
WORKING_DIR = Path(__file__).parent
CACHE_BUILDER_SCRIPT = WORKING_DIR / "cache_builder.py"
LOG_FILE = WORKING_DIR / "location_manager.log"
BIRDNET_CONFIG_PATH = "/root/birdnet-go-app/config/config.yaml"

# Significant distance threshold (km) - update if location changed by this much
LOCATION_CHANGE_THRESHOLD_KM = 100

# --- Logging Setup ---
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler(LOG_FILE),
        logging.StreamHandler(sys.stdout)
    ]
)
logger = logging.getLogger(__name__)

# --- Helper Functions ---
def run_cache_builder(flags: list) -> int:
    """Run cache_builder.py with specified flags.

    Args:
        flags: List of command-line flags (e.g., ['--update-species', '--incremental'])

    Returns:
        Exit code from cache_builder.py
    """
    try:
        cmd = ["python3", str(CACHE_BUILDER_SCRIPT)] + flags
        logger.info(f"Running cache builder: {' '.join(cmd)}")

        result = subprocess.run(
            cmd,
            cwd=str(WORKING_DIR),
            capture_output=True,
            text=True
        )

        # Log output
        if result.stdout:
            for line in result.stdout.splitlines():
                logger.info(f"  [cache_builder] {line}")

        if result.stderr:
            for line in result.stderr.splitlines():
                logger.warning(f"  [cache_builder] {line}")

        logger.info(f"Cache builder exit code: {result.returncode}")
        return result.returncode

    except Exception as e:
        logger.error(f"Failed to run cache builder: {e}")
        return 1

def check_birdnet_go_running() -> bool:
    """Check if BirdNET-Go service is running and accessible."""
    try:
        import requests
        # Try the API endpoint instead of /health (which doesn't exist)
        response = requests.get("http://localhost:8080/api/v2/detections/recent", timeout=5)
        return response.status_code == 200
    except Exception:
        return False

def wait_for_birdnet_go(max_wait_seconds: int = 30) -> bool:
    """Wait for BirdNET-Go to become available.

    Args:
        max_wait_seconds: Maximum time to wait

    Returns:
        True if BirdNET-Go became available, False if timeout
    """
    import time
    logger.info(f"Waiting for BirdNET-Go to become available (max {max_wait_seconds}s)...")

    for i in range(max_wait_seconds):
        if check_birdnet_go_running():
            logger.info(f"BirdNET-Go is available (waited {i+1}s)")
            return True
        time.sleep(1)

    logger.warning(f"BirdNET-Go did not become available after {max_wait_seconds}s")
    return False

# --- Main Logic ---
def main():
    """Main location manager workflow."""
    logger.info("=" * 60)
    logger.info("BirdNET Display - Location Manager Starting")
    logger.info("=" * 60)

    try:
        # Step 1: Wait for BirdNET-Go to be available
        if not wait_for_birdnet_go():
            logger.error("BirdNET-Go is not available. Cannot proceed.")
            logger.info("Location manager will exit. Manual intervention required.")
            return 1

        # Step 2: Load configuration manager
        logger.info(f"Loading BirdNET-Go configuration from {BIRDNET_CONFIG_PATH}")
        config_manager = BirdNETConfigManager(BIRDNET_CONFIG_PATH)

        if not config_manager.load():
            logger.error("Failed to load BirdNET-Go configuration")
            return 1

        # Step 3: Get current location from config
        current_location = config_manager.get_location()
        if current_location:
            logger.info(f"Current location in config: "
                       f"Lat {current_location['latitude']}, "
                       f"Lon {current_location['longitude']}")
        else:
            logger.info("No valid location configured (0,0 or not set)")

        # Step 4: Detect new location
        logger.info("Detecting current location...")
        location_detector = LocationDetector()
        detected_location = location_detector.detect_location()

        if not detected_location:
            logger.warning("Could not detect location using any method")

            # If no current location either, this is an error state
            if not current_location:
                logger.error("No location available and detection failed. Exiting.")
                logger.info("Consider creating a manual location config file.")
                logger.info("Run: python3 -c \"from utils.geolocation import create_manual_config_template; create_manual_config_template()\"")
                return 1

            # Keep existing location
            logger.info("Keeping existing location configuration")
            detected_location = current_location
        else:
            logger.info(f"Detected location: "
                       f"Lat {detected_location['latitude']}, "
                       f"Lon {detected_location['longitude']} "
                       f"(Method: {detected_location.get('source', 'unknown')})")

        # Step 5: Check if location changed significantly
        location_changed = False
        if current_location:
            location_changed = config_manager.location_changed(
                detected_location['latitude'],
                detected_location['longitude'],
                threshold_km=LOCATION_CHANGE_THRESHOLD_KM
            )

            if location_changed:
                distance = config_manager.get_location_distance(
                    detected_location['latitude'],
                    detected_location['longitude']
                )
                logger.info(f"Location changed by {distance:.1f} km (threshold: {LOCATION_CHANGE_THRESHOLD_KM} km)")
            else:
                logger.info(f"Location change is within threshold ({LOCATION_CHANGE_THRESHOLD_KM} km)")
        else:
            # No current location means this is a new setup
            location_changed = True
            logger.info("First-time location setup")

        # Step 6: Update configuration if needed
        if location_changed:
            logger.info("Updating BirdNET-Go configuration with new location...")

            if config_manager.set_location(
                detected_location['latitude'],
                detected_location['longitude']
            ):
                logger.info("Configuration updated successfully")

                if config_manager.save(create_backup=True):
                    logger.info("Configuration saved with backup")
                else:
                    logger.error("Failed to save configuration")
                    return 1
            else:
                logger.error("Failed to update location in configuration")
                return 1

            # Step 7: Update species list from API
            logger.info("Fetching updated species list from BirdNET-Go API...")
            exit_code = run_cache_builder(['--update-species', '--incremental', '--yes'])

            if exit_code == 1:
                logger.error("Failed to update species list and cache")
                return 1
            elif exit_code == 2:
                logger.info("Species list updated, cache already complete")
            else:
                logger.info("Species list and cache updated successfully")

            logger.info("Location update complete")
            return 0

        else:
            # Step 8: Location unchanged, check cache status
            logger.info("Location unchanged, checking cache status...")
            exit_code = run_cache_builder(['--check-only'])

            if exit_code == 1:
                logger.warning("Cache check failed, attempting incremental update...")
                exit_code = run_cache_builder(['--incremental'])

                if exit_code == 0:
                    logger.info("Cache synchronized successfully")
                    return 0
                elif exit_code == 2:
                    logger.info("Cache is up to date")
                    return 2
                else:
                    logger.error("Cache synchronization failed")
                    return 1

            elif exit_code == 2:
                logger.info("Cache is up to date, no changes needed")
                return 2

            else:
                logger.info("Cache status check completed")
                return 0

    except Exception as e:
        logger.exception(f"Unexpected error in location manager: {e}")
        return 1

    finally:
        logger.info("=" * 60)
        logger.info("Location Manager Finished")
        logger.info("=" * 60)


if __name__ == '__main__':
    exit_code = main()
    sys.exit(exit_code)
