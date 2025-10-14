# BirdNET Display

A Python-based web application designed to run on a Raspberry Pi alongside BirdNET-Go. It displays the latest bird detections on a screen attached to the Pi, using BirdNET data and local image caches. Designed for standard 800x480px screens.

> **Note**: This display component is part of the [BirdNET-Gone](https://github.com/festion/birdnet-gone) unified repository. For the main BirdNET-Go application, see the [parent README](../README.md).

## Features
- Designed for Raspberry Pi with a connected display
- Integrates with BirdNET-Go to show the latest bird detections
- Displays the IP address (including a QR code) of the Raspberry Pi on the webpage
- Caches images for all birds in the species list so the app can work completely offline
- Simple and responsive web interface
- Kiosk mode for dedicated display on a Raspberry Pi
- System controls from the web interface (brightness, reboot, power off)
- Microphone status indicator that pings an ESP32 RTSP stream to verify audio connection

## Prerequisites

This project requires a locally installed and running instance of BirdNET-Go. You can install it by running:

```bash
curl -fsSL https://github.com/festion/birdnet-gone/raw/main/install.sh -o install.sh
bash ./install.sh
```

## Setup and Installation

### Automatic Installation (Recommended for Raspberry Pi)

The `install.sh` script automates the entire setup process on a Raspberry Pi.

1. **Clone the repository:**
   ```bash
   git clone https://github.com/festion/birdnet-gone.git
   cd birdnet-gone/display
   ```

2. **Run the installer:**
   ```bash
   chmod +x install.sh
   ./install.sh
   ```

   The script will:
   - Create an installation directory (`~/birdnet_display`)
   - Set up a Python virtual environment
   - Install all required dependencies
   - Build the initial image cache from your `species_list.csv`
   - Optionally configure kiosk mode and BirdNET-Go networking

### Manual Installation

1. **Navigate to the display directory:**
   ```bash
   cd birdnet-gone/display
   ```

2. **Create a Python virtual environment:**
   ```bash
   python3 -m venv venv
   source venv/bin/activate
   ```

3. **Install the required Python packages:**
   ```bash
   pip install -r requirements.txt
   ```

4. **Build the image cache:**
   ```bash
   python cache_builder.py
   ```

5. **Run the application:**
   ```bash
   python birdnet_display.py
   ```

## Usage

- **To run the application manually:**
  ```bash
  cd ~/birdnet_display
  ./run.sh
  ```
- If you enabled kiosk mode during installation, the application will start automatically on boot
- The Flask app serves the web interface on port 5000 at `http://<your-pi-ip>:5000`

## Configuration

### Species List

To customize the birds displayed in offline mode, edit the `species_list.csv` file. Add the common and scientific names of the birds you want to see.

```csv
Common Name,Scientific Name
Australian Magpie,Gymnorhina tibicen
Torresian Crow,Corvus orru
...
```

After modifying the list, rebuild the cache:

```bash
cd ~/birdnet_display
source venv/bin/activate
python cache_builder.py
```

### Application Settings

The main application settings are at the top of `birdnet_display.py`:

- `BASE_URL`: The URL of your BirdNET-Go instance
- `SERVER_PORT`: The port for the display web server

## Docker Deployment

You can also run the display component using Docker Compose (recommended for unified deployment):

```bash
cd birdnet-gone
docker-compose up -d
```

This will start both BirdNET-Go and the display interface together.

## License

MIT License (Display component only)

For the main BirdNET-Go license, see [../LICENSE](../LICENSE)
