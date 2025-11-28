#!/bin/bash
# Add a delay to allow the desktop and network to fully initialize
sleep 15
# Launch Chromium with Wayland and disable keyring prompt
/usr/bin/chromium --ozone-platform=wayland --password-store=basic --noerrdialogs --disable-infobars --kiosk http://localhost:5000
