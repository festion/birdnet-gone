"""
Geolocation Detection Module
Provides multiple methods for determining device location with fallback chain.
"""

import requests
import json
import os
import subprocess
from typing import Optional, Dict, Tuple

# Color codes for terminal output
GREEN = '\033[0;32m'
YELLOW = '\033[1;33m'
RED = '\033[0;31m'
NC = '\033[0m'  # No Color


class LocationDetector:
    """Detects device location using multiple methods with fallback chain."""
    
    def __init__(self, timeout: int = 10):
        """
        Initialize location detector.
        
        Args:
            timeout: Timeout in seconds for API requests
        """
        self.timeout = timeout
        self.session = requests.Session()
        self.session.headers.update({
            'User-Agent': 'Mozilla/5.0 (compatible; BirdNET-Display/1.0)'
        })
    
    def detect_location(self) -> Optional[Dict[str, any]]:
        """
        Detect location using multiple methods with fallback chain.
        
        Returns:
            Dict with keys: latitude, longitude, method, description, accuracy
            None if all methods fail
        """
        # Method 1: IP Geolocation (primary)
        print(f"{YELLOW}[Location] Attempting IP geolocation...{NC}")
        location = self._try_ip_geolocation()
        if location:
            return location
        
        # Method 2: GPS Hardware (secondary)
        print(f"{YELLOW}[Location] Checking for GPS hardware...{NC}")
        location = self._try_gps_hardware()
        if location:
            return location
        
        # Method 3: Manual config file (tertiary)
        print(f"{YELLOW}[Location] Checking for manual configuration...{NC}")
        location = self._try_manual_config()
        if location:
            return location
        
        print(f"{RED}[Location] All detection methods failed{NC}")
        return None
    
    def _try_ip_geolocation(self) -> Optional[Dict[str, any]]:
        """
        Try multiple IP geolocation APIs with fallback.
        
        Returns:
            Location dict or None
        """
        # API list in priority order
        apis = [
            {
                'url': 'https://ipapi.co/json/',
                'parser': self._parse_ipapi_co
            },
            {
                'url': 'http://ip-api.com/json/',
                'parser': self._parse_ip_api_com
            },
            {
                'url': 'https://ipinfo.io/json',
                'parser': self._parse_ipinfo_io
            }
        ]
        
        for api in apis:
            try:
                response = self.session.get(api['url'], timeout=self.timeout)
                response.raise_for_status()
                data = response.json()
                location = api['parser'](data)
                if location and self._validate_coordinates(location['latitude'], location['longitude']):
                    print(f"{GREEN}[Location] ✓ Detected via {location['method']}: "
                          f"{location['description']} ({location['latitude']:.4f}, {location['longitude']:.4f}){NC}")
                    return location
            except (requests.exceptions.RequestException, json.JSONDecodeError, KeyError) as e:
                print(f"{YELLOW}[Location] API {api['url']} failed: {e}{NC}")
                continue
        
        return None
    
    def _parse_ipapi_co(self, data: Dict) -> Optional[Dict[str, any]]:
        """Parse ipapi.co response."""
        if 'error' in data:
            return None
        
        return {
            'latitude': float(data['latitude']),
            'longitude': float(data['longitude']),
            'method': 'ipapi.co',
            'description': f"{data.get('city', 'Unknown')}, {data.get('region', '')}, {data.get('country_name', 'Unknown')}".strip(', '),
            'accuracy': 'city-level (~10-50km)'
        }
    
    def _parse_ip_api_com(self, data: Dict) -> Optional[Dict[str, any]]:
        """Parse ip-api.com response."""
        if data.get('status') != 'success':
            return None
        
        return {
            'latitude': float(data['lat']),
            'longitude': float(data['lon']),
            'method': 'ip-api.com',
            'description': f"{data.get('city', 'Unknown')}, {data.get('regionName', '')}, {data.get('country', 'Unknown')}".strip(', '),
            'accuracy': 'city-level (~10-50km)'
        }
    
    def _parse_ipinfo_io(self, data: Dict) -> Optional[Dict[str, any]]:
        """Parse ipinfo.io response."""
        if 'loc' not in data:
            return None
        
        lat, lon = data['loc'].split(',')
        return {
            'latitude': float(lat),
            'longitude': float(lon),
            'method': 'ipinfo.io',
            'description': f"{data.get('city', 'Unknown')}, {data.get('region', '')}, {data.get('country', 'Unknown')}".strip(', '),
            'accuracy': 'city-level (~10-50km)'
        }
    
    def _try_gps_hardware(self) -> Optional[Dict[str, any]]:
        """
        Try to get location from GPS hardware via gpsd.
        
        Returns:
            Location dict or None
        """
        try:
            # Check if gpsd is running
            result = subprocess.run(
                ['systemctl', 'is-active', 'gpsd'],
                capture_output=True,
                text=True,
                timeout=5
            )
            
            if result.returncode != 0:
                print(f"{YELLOW}[Location] gpsd service not running{NC}")
                return None
            
            # Try to get position from gpsd
            # This requires python-gps package
            try:
                import gps
                session = gps.gps(mode=gps.WATCH_ENABLE)
                
                # Try to get a fix (wait up to 30 seconds)
                for _ in range(30):
                    report = session.next()
                    if report['class'] == 'TPV':
                        if hasattr(report, 'lat') and hasattr(report, 'lon'):
                            if self._validate_coordinates(report.lat, report.lon):
                                print(f"{GREEN}[Location] ✓ Detected via GPS hardware{NC}")
                                return {
                                    'latitude': report.lat,
                                    'longitude': report.lon,
                                    'method': 'GPS hardware',
                                    'description': f'GPS Fix ({report.lat:.6f}, {report.lon:.6f})',
                                    'accuracy': 'high-precision (~5-10m)'
                                }
            except ImportError:
                print(f"{YELLOW}[Location] python-gps package not installed{NC}")
            except Exception as e:
                print(f"{YELLOW}[Location] GPS read failed: {e}{NC}")
        
        except (subprocess.TimeoutExpired, subprocess.CalledProcessError, FileNotFoundError):
            pass
        
        return None
    
    def _try_manual_config(self) -> Optional[Dict[str, any]]:
        """
        Try to load location from manual configuration file.
        
        Returns:
            Location dict or None
        """
        config_paths = [
            'location_config.json',
            '/home/jeremy/birdnet_display/location_config.json',
            os.path.expanduser('~/birdnet_display/location_config.json')
        ]
        
        for path in config_paths:
            if os.path.exists(path):
                try:
                    with open(path, 'r') as f:
                        config = json.load(f)
                    
                    if 'location' in config:
                        loc = config['location']
                        lat = loc.get('latitude')
                        lon = loc.get('longitude')
                        
                        if lat is not None and lon is not None:
                            if self._validate_coordinates(lat, lon):
                                print(f"{GREEN}[Location] ✓ Using manual configuration from {path}{NC}")
                                return {
                                    'latitude': float(lat),
                                    'longitude': float(lon),
                                    'method': 'manual configuration',
                                    'description': loc.get('description', f'Manual ({lat:.4f}, {lon:.4f})'),
                                    'accuracy': 'user-specified'
                                }
                except (IOError, json.JSONDecodeError, KeyError, ValueError) as e:
                    print(f"{YELLOW}[Location] Failed to read config {path}: {e}{NC}")
                    continue
        
        return None
    
    def _validate_coordinates(self, lat: float, lon: float) -> bool:
        """
        Validate that coordinates are within valid ranges and not default values.
        
        Args:
            lat: Latitude
            lon: Longitude
        
        Returns:
            True if valid, False otherwise
        """
        # Check valid ranges
        if not (-90 <= lat <= 90 and -180 <= lon <= 180):
            return False
        
        # Check not default/null island (0, 0)
        if lat == 0 and lon == 0:
            return False
        
        return True
    
    def format_location_info(self, location: Dict[str, any]) -> str:
        """
        Format location info as human-readable string.
        
        Args:
            location: Location dict from detect_location()
        
        Returns:
            Formatted string
        """
        return (
            f"Location: {location['description']}\n"
            f"Coordinates: {location['latitude']:.6f}, {location['longitude']:.6f}\n"
            f"Detection Method: {location['method']}\n"
            f"Accuracy: {location['accuracy']}"
        )


def create_manual_config_template(output_path: str = 'location_config.json'):
    """
    Create a template manual configuration file.
    
    Args:
        output_path: Where to save the template
    """
    template = {
        "location": {
            "latitude": 33.7490,
            "longitude": -84.3880,
            "description": "Atlanta, GA",
            "method": "manual"
        },
        "cache": {
            "auto_update": True,
            "update_interval_days": 7,
            "force_update_on_location_change": True
        },
        "preferences": {
            "preferred_detection_method": "ip_geolocation",
            "fallback_to_auto_detection": True
        }
    }
    
    try:
        with open(output_path, 'w') as f:
            json.dump(template, f, indent=2)
        print(f"{GREEN}[Config] Created template configuration at {output_path}{NC}")
        print(f"{YELLOW}[Config] Please edit this file with your actual coordinates{NC}")
        return True
    except IOError as e:
        print(f"{RED}[Config] Failed to create template: {e}{NC}")
        return False


if __name__ == '__main__':
    # Test the location detector
    print("Testing Location Detector...\n")
    detector = LocationDetector()
    location = detector.detect_location()
    
    if location:
        print(f"\n{GREEN}=== Location Detection Successful ==={NC}")
        print(detector.format_location_info(location))
    else:
        print(f"\n{RED}=== Location Detection Failed ==={NC}")
        print("Consider creating a manual configuration file:")
        create_manual_config_template()
