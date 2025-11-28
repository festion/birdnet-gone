"""
Configuration Manager Module
Handles reading and writing BirdNET-Go YAML configuration files.
"""

import yaml
import os
import shutil
from typing import Optional, Dict, Any
from datetime import datetime

# Color codes for terminal output
GREEN = '\033[0;32m'
YELLOW = '\033[1;33m'
RED = '\033[0;31m'
NC = '\033[0m'  # No Color


class BirdNETConfigManager:
    """Manages BirdNET-Go configuration file (config.yaml)."""
    
    def __init__(self, config_path: str = '/root/birdnet-go-app/config/config.yaml'):
        """
        Initialize config manager.
        
        Args:
            config_path: Path to BirdNET-Go config.yaml file
        """
        self.config_path = config_path
        self.backup_path = f"{config_path}.backup"
        self.config_data = None
    
    def load(self) -> bool:
        """
        Load configuration from file.
        
        Returns:
            True if successful, False otherwise
        """
        if not os.path.exists(self.config_path):
            print(f"{RED}[Config] Configuration file not found: {self.config_path}{NC}")
            return False
        
        try:
            with open(self.config_path, 'r') as f:
                self.config_data = yaml.safe_load(f)
            print(f"{GREEN}[Config] ✓ Loaded configuration from {self.config_path}{NC}")
            return True
        except (IOError, yaml.YAMLError) as e:
            print(f"{RED}[Config] Failed to load configuration: {e}{NC}")
            return False
    
    def save(self, create_backup: bool = True) -> bool:
        """
        Save configuration to file.
        
        Args:
            create_backup: Whether to create a backup before saving
        
        Returns:
            True if successful, False otherwise
        """
        if self.config_data is None:
            print(f"{RED}[Config] No configuration data to save{NC}")
            return False
        
        try:
            # Create backup if requested
            if create_backup and os.path.exists(self.config_path):
                timestamp = datetime.now().strftime('%Y%m%d_%H%M%S')
                backup_path = f"{self.config_path}.backup_{timestamp}"
                shutil.copy2(self.config_path, backup_path)
                print(f"{GREEN}[Config] ✓ Created backup: {backup_path}{NC}")
            
            # Write configuration
            with open(self.config_path, 'w') as f:
                yaml.dump(self.config_data, f, default_flow_style=False, sort_keys=False)
            
            print(f"{GREEN}[Config] ✓ Saved configuration to {self.config_path}{NC}")
            return True
        except (IOError, yaml.YAMLError) as e:
            print(f"{RED}[Config] Failed to save configuration: {e}{NC}")
            return False
    
    def get_location(self) -> Optional[Dict[str, float]]:
        """
        Get current location from configuration.
        
        Returns:
            Dict with 'latitude' and 'longitude' keys, or None if not set/invalid
        """
        if self.config_data is None:
            return None
        
        birdnet_section = self.config_data.get('birdnet', {})
        lat = birdnet_section.get('latitude')
        lon = birdnet_section.get('longitude')
        
        # Validate coordinates
        if lat is None or lon is None:
            return None
        
        if not isinstance(lat, (int, float)) or not isinstance(lon, (int, float)):
            return None
        
        # Check if set to default (0, 0) or invalid range
        if (lat == 0 and lon == 0) or not (-90 <= lat <= 90 and -180 <= lon <= 180):
            return None
        
        return {'latitude': float(lat), 'longitude': float(lon)}
    
    def set_location(self, latitude: float, longitude: float) -> bool:
        """
        Set location in configuration.
        
        Args:
            latitude: Latitude coordinate
            longitude: Longitude coordinate
        
        Returns:
            True if successful, False otherwise
        """
        if self.config_data is None:
            print(f"{RED}[Config] Configuration not loaded{NC}")
            return False
        
        # Validate coordinates
        if not (-90 <= latitude <= 90 and -180 <= longitude <= 180):
            print(f"{RED}[Config] Invalid coordinates: {latitude}, {longitude}{NC}")
            return False
        
        # Ensure birdnet section exists
        if 'birdnet' not in self.config_data:
            self.config_data['birdnet'] = {}
        
        # Update location
        self.config_data['birdnet']['latitude'] = latitude
        self.config_data['birdnet']['longitude'] = longitude
        
        print(f"{GREEN}[Config] ✓ Updated location: {latitude:.6f}, {longitude:.6f}{NC}")
        return True
    
    def location_changed(self, new_latitude: float, new_longitude: float, threshold_km: float = 100) -> bool:
        """
        Check if new location differs significantly from current location.
        
        Args:
            new_latitude: New latitude
            new_longitude: New longitude
            threshold_km: Distance threshold in kilometers
        
        Returns:
            True if location changed significantly, False otherwise
        """
        current_location = self.get_location()
        if current_location is None:
            return True  # No location set, so this is a change
        
        # Calculate distance using Haversine formula
        from math import radians, sin, cos, sqrt, atan2
        
        lat1 = radians(current_location['latitude'])
        lon1 = radians(current_location['longitude'])
        lat2 = radians(new_latitude)
        lon2 = radians(new_longitude)
        
        dlat = lat2 - lat1
        dlon = lon2 - lon1
        
        a = sin(dlat/2)**2 + cos(lat1) * cos(lat2) * sin(dlon/2)**2
        c = 2 * atan2(sqrt(a), sqrt(1-a))
        
        # Earth radius in kilometers
        radius = 6371
        distance = radius * c
        
        if distance > threshold_km:
            print(f"{YELLOW}[Config] Location changed by {distance:.1f} km (threshold: {threshold_km} km){NC}")
            return True
        else:
            print(f"{GREEN}[Config] Location unchanged ({distance:.1f} km difference){NC}")
            return False
    
    def get_setting(self, *keys) -> Optional[Any]:
        """
        Get a setting from configuration using dot notation.
        
        Args:
            *keys: Path to setting (e.g., 'birdnet', 'threshold')
        
        Returns:
            Setting value or None if not found
        """
        if self.config_data is None:
            return None
        
        current = self.config_data
        for key in keys:
            if isinstance(current, dict) and key in current:
                current = current[key]
            else:
                return None
        
        return current
    
    def set_setting(self, value: Any, *keys) -> bool:
        """
        Set a setting in configuration using dot notation.
        
        Args:
            value: Value to set
            *keys: Path to setting (e.g., 'birdnet', 'threshold')
        
        Returns:
            True if successful, False otherwise
        """
        if self.config_data is None:
            return False
        
        if len(keys) == 0:
            return False
        
        # Navigate to parent dict
        current = self.config_data
        for key in keys[:-1]:
            if key not in current:
                current[key] = {}
            current = current[key]
        
        # Set value
        current[keys[-1]] = value
        return True
    
    def format_config_summary(self) -> str:
        """
        Get a human-readable summary of key configuration settings.
        
        Returns:
            Formatted string
        """
        if self.config_data is None:
            return "Configuration not loaded"
        
        lines = []
        lines.append("=== BirdNET-Go Configuration ===")
        
        # Location
        location = self.get_location()
        if location:
            lines.append(f"Location: {location['latitude']:.6f}, {location['longitude']:.6f}")
        else:
            lines.append("Location: Not set (0.0, 0.0)")
        
        # Detection settings
        threshold = self.get_setting('birdnet', 'threshold')
        sensitivity = self.get_setting('birdnet', 'sensitivity')
        locale = self.get_setting('birdnet', 'locale')
        
        if threshold is not None:
            lines.append(f"Confidence Threshold: {threshold}")
        if sensitivity is not None:
            lines.append(f"Sensitivity: {sensitivity}")
        if locale is not None:
            lines.append(f"Locale: {locale}")
        
        # Audio source
        audio_source = self.get_setting('realtime', 'audio', 'source')
        if audio_source:
            lines.append(f"Audio Source: {audio_source}")
        
        return "\n".join(lines)


def restore_from_backup(config_path: str) -> bool:
    """
    Restore configuration from most recent backup.
    
    Args:
        config_path: Path to config file
    
    Returns:
        True if successful, False otherwise
    """
    # Find most recent backup
    backup_dir = os.path.dirname(config_path)
    backup_basename = os.path.basename(config_path) + '.backup_'
    
    try:
        backups = [
            f for f in os.listdir(backup_dir)
            if f.startswith(backup_basename)
        ]
        
        if not backups:
            print(f"{RED}[Config] No backups found{NC}")
            return False
        
        # Sort by timestamp (filename format: config.yaml.backup_YYYYMMDD_HHMMSS)
        backups.sort(reverse=True)
        most_recent = os.path.join(backup_dir, backups[0])
        
        # Restore
        shutil.copy2(most_recent, config_path)
        print(f"{GREEN}[Config] ✓ Restored from backup: {most_recent}{NC}")
        return True
    
    except (IOError, OSError) as e:
        print(f"{RED}[Config] Failed to restore from backup: {e}{NC}")
        return False


if __name__ == '__main__':
    # Test the config manager
    print("Testing BirdNET Config Manager...\n")
    
    # Try to load config
    manager = BirdNETConfigManager('/root/birdnet-go-app/config/config.yaml')
    
    if manager.load():
        print("\n" + manager.format_config_summary())
        
        current_location = manager.get_location()
        if current_location:
            print(f"\nCurrent location: {current_location}")
        else:
            print("\nNo valid location set")
    else:
        print("Failed to load configuration")
