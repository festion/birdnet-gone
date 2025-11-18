# ESP32-CAM Hardware Guide for Audio Streaming

## 📌 ESP32-CAM Overview

The ESP32-CAM is a compact development board featuring:
- **ESP32-S chip** (dual-core 240MHz)
- **4MB PSRAM** (for buffering)
- **4MB Flash** (firmware storage)
- **OV2640 Camera** (not used for audio project)
- **MicroSD card slot** (optional future use)
- **Built-in WiFi/Bluetooth**

### ESP32-CAM Pinout

```
                         ┌─────────────┐
                    3V3  │1          2 │ GND
                   IO16  │3          4 │ IO0 (BOOT)
                    U0R  │5          6 │ GND
                    U0T  │7          8 │ 5V
              IO15 (SCK) │9         10 │ IO14 (WS)
              IO13 (SD)  │11        12 │ IO12
                    GND  │13        14 │ IO2 (Flash LED)
                   IO4   │15        16 │ GND
                         └─────────────┘
```

### GPIO Pin Availability

**✅ Safe for I2S (not used by camera):**
- GPIO13, GPIO14, GPIO15 ← **Used for I2S**
- GPIO33 (LED - can be used if LED disabled)

**❌ Avoid (used by camera/SD card):**
- GPIO0-12 (camera interface, boot pins)
- GPIO16 (PSRAM)
- GPIO2 (flash LED, boot strapping)

## 🎤 I2S Microphone Modules

### Recommended: INMP441

**Specifications:**
- Ultra-low noise: 61 dB SNR
- High PSR: -75 dBFS
- Wide frequency range: 60Hz - 15kHz
- I2S digital output (24-bit)
- 3.3V operation
- Omnidirectional pattern

**Pinout:**
```
  ┌─────────────┐
  │   INMP441   │
  │  ┌───────┐  │
  │  │  MIC  │  │
  │  └───────┘  │
  │             │
  │ VDD L/R GND │
  │ WS SCK SD   │
  └─────────────┘
```

| Pin | Function | Connect To |
|-----|----------|------------|
| VDD | Power (1.8-3.3V) | ESP32 3V3 |
| L/R | Channel Select | GND (left) or VDD (right) |
| GND | Ground | ESP32 GND |
| WS  | Word Select (LRCLK) | ESP32 GPIO14 |
| SCK | Serial Clock (BCLK) | ESP32 GPIO15 |
| SD  | Serial Data (DOUT) | ESP32 GPIO13 |

**Wiring Notes:**
- ✅ **Correct polarity critical** - reversed power will damage mic
- ✅ **L/R must be tied** - floating causes intermittent operation
- ✅ **Keep data wires short** - <10cm to minimize noise
- ✅ **Twist WS/SD together** - reduces crosstalk

**Where to Buy:**
- Amazon: ~$7 for 2-pack
- AliExpress: ~$3 for 2-pack
- SparkFun: ~$12 (official board)

### Alternative: SPH0645LM4H

**Specifications:**
- Good SNR: 65 dB(A)
- I2S digital output (18-bit)
- Wide frequency range: 50Hz - 15kHz
- 3.3V operation
- Omnidirectional pattern

**Pinout:**
```
  ┌─────────────┐
  │  SPH0645    │
  │  ┌───────┐  │
  │  │  MIC  │  │
  │  └───────┘  │
  │             │
  │ 3V BCLK GND │
  │ SEL DOUT WS │
  └─────────────┘
```

| Pin | Function | Connect To |
|-----|----------|------------|
| 3V  | Power | ESP32 3V3 |
| SEL | Channel Select | GND (left) or 3V (right) |
| GND | Ground | ESP32 GND |
| BCLK | Bit Clock | ESP32 GPIO15 (SCK) |
| DOUT | Data Out | ESP32 GPIO13 (SD) |
| WS   | Word Select | ESP32 GPIO14 (WS) |

**Key Differences from INMP441:**
- Slightly better SNR (65 vs 61 dB)
- Uses different pin names but same I2S protocol
- Adafruit breakout includes level shifter (not needed for 3.3V)

### Budget Option: ICS-43434

**Specifications:**
- Decent SNR: 63 dB
- I2S digital output
- 3.3V operation
- Lower cost

**⚠️ WARNING - VERIFY PINOUT:**

Some ICS-43434 modules (especially QUAINTBYTE) have **mislabeled pins** where WS and SD are swapped!

**Standard pinout:**
```
VDD - L/R - GND - WS - SCK - SD
```

**QUAINTBYTE pinout (physical pins swapped):**
```
VDD - L/R - GND - SD(label) - SCK - WS(label)
                   ^^^actual WS      ^^^actual SD
```

**Testing for pin swap:**
1. Wire according to labels
2. Check audio quality (should be >90% non-zero samples)
3. If mostly silence (>99% zeros), try swapping WS and SD in firmware
4. Update `src/main.cpp` lines 68-70:
   ```cpp
   #define I2S_WS_PIN      13  // Swapped from 14
   #define I2S_SD_PIN      14  // Swapped from 13
   ```

## 🔌 Complete Wiring Guide

### Step-by-Step Connection

**Tools needed:**
- Soldering iron and solder
- Wire strippers
- Multimeter (for continuity testing)
- 22-26 AWG hookup wire (6 colors recommended)

**Recommended wire colors:**
- Red: 3.3V power
- Black: Ground
- Yellow: WS (Word Select)
- Green: SCK (Serial Clock)
- Blue: SD (Serial Data)
- Black/Brown: L/R (Channel Select)

### Wiring Procedure

1. **Prepare ESP32-CAM**:
   - No soldering needed (use existing pins)
   - Identify GPIO13, GPIO14, GPIO15, 3V3, GND

2. **Prepare INMP441 Module**:
   - Module usually comes with header pins
   - Solder header pins if needed (straight, not angled)

3. **Cut and Strip Wires**:
   - Cut 6 wires to 8-10cm length
   - Strip 3mm from each end
   - Tin wire ends with solder

4. **Solder to ESP32-CAM**:
   ```
   GPIO13 ────► Blue wire (SD)
   GPIO14 ────► Yellow wire (WS)
   GPIO15 ────► Green wire (SCK)
   3V3    ────► Red wire (VDD)
   GND    ────► Black wire (GND)
   GND    ────► Black/Brown wire (L/R)
   ```

5. **Solder to INMP441**:
   ```
   SD  ◄──── Blue wire
   WS  ◄──── Yellow wire
   SCK ◄──── Green wire
   VDD ◄──── Red wire
   GND ◄──── Black wire
   L/R ◄──── Black/Brown wire (or connect L/R to GND pin on module)
   ```

6. **Verify Connections**:
   - Use multimeter in continuity mode
   - Check each connection: ESP32 pin → Microphone pin
   - Verify no shorts between adjacent pins
   - Check power: 3.3V between VDD and GND (when powered)

### Connection Verification Checklist

Before powering on:
- [ ] VDD connected to 3V3 (not 5V!)
- [ ] GND connected to GND
- [ ] L/R connected to GND or VDD (not floating)
- [ ] WS connected to GPIO14
- [ ] SCK connected to GPIO15
- [ ] SD connected to GPIO13
- [ ] No shorts between power and ground
- [ ] All solder joints clean and shiny

## 🔋 Power Considerations

### Power Requirements

**ESP32-CAM:**
- Idle: ~100mA @ 5V
- WiFi active: ~200mA @ 5V
- Streaming: ~300mA @ 5V
- Peak: ~400mA @ 5V (WiFi TX burst)

**INMP441 Microphone:**
- Typical: ~1.4mA @ 3.3V
- Peak: ~2mA @ 3.3V

**Total System:**
- Average: ~300mA @ 5V
- Peak: ~400mA @ 5V
- Minimum power supply: **1A @ 5V recommended**

### Power Supply Options

#### Option 1: ESP32-CAM-MB (Recommended for Flashing)

```
USB Cable → ESP32-CAM-MB → ESP32-CAM
                        → 5V regulated output
```

**Pros:**
- Stable 5V power
- Integrated USB-to-serial (CH340)
- Easy firmware flashing
- Reset/boot buttons

**Cons:**
- Larger footprint
- ESP32-CAM must stay inserted
- Not ideal for permanent installation

#### Option 2: USB Power Adapter (Standalone Operation)

```
Wall Adapter (5V 1A) → ESP32-CAM 5V pin
```

**Pros:**
- Compact
- No programmer board needed after initial flash
- Good for permanent installation

**Cons:**
- Need FTDI adapter for firmware updates (or use OTA)
- Manual boot mode for flashing

#### Option 3: Battery Power (Portable)

```
18650 Battery (3.7V) → Boost Converter (5V) → ESP32-CAM
```

**Example:**
- 18650 battery: 3000mAh @ 3.7V
- Boost converter: 90% efficiency
- Runtime: ~8-10 hours continuous streaming

**Pros:**
- Portable
- No wires
- Good for field deployment

**Cons:**
- Needs charging circuit
- Limited runtime
- Voltage monitoring required

#### Option 4: Solar + Battery (Long-term)

```
Solar Panel (6V 2W) → Charge Controller → Battery → ESP32-CAM
```

**Example setup:**
- 6V 2W solar panel
- TP4056 charge controller
- 18650 battery (3.7V 3000mAh)
- MT3608 boost converter (to 5V)

**Pros:**
- Indefinite operation (weather permitting)
- No maintenance

**Cons:**
- Complex circuit
- Weather dependent
- Higher cost

### Power Quality

**Important:**
- ESP32 is **sensitive to voltage drops** during WiFi transmission
- Use **capacitors** for stable power:
  - 100µF electrolytic across 5V/GND (close to ESP32)
  - 0.1µF ceramic across 3.3V/GND (close to microphone)
- Avoid **long thin power wires** (voltage drop)
- Ensure **stable 5V supply** (measure with multimeter under load)

## 🛠️ Assembly Tips

### Soldering Best Practices

1. **Temperature**: 350°C for lead-free, 320°C for leaded
2. **Tip**: Clean, tinned (not oxidized)
3. **Technique**: Heat pad + pin, then apply solder
4. **Duration**: 2-3 seconds per joint
5. **Inspection**: Shiny, volcano-shaped joint (not dull/ball)

### Wire Management

- **Strain relief**: Add hot glue at solder points
- **Labeling**: Mark wires with tape labels
- **Routing**: Keep I2S wires away from power wires
- **Length**: Short as practical (<15cm total)

### Enclosure Recommendations

**For indoor use:**
- 3D printed case (STL files available on Thingiverse)
- Small project box with mounting holes
- Ensure microphone has clear path to ambient sound

**For outdoor use:**
- IP65+ rated enclosure
- Microphone port with acoustic mesh (waterproof but breathable)
- Desiccant pack for moisture control
- UV-resistant materials

## 🧪 Testing and Validation

### Initial Power-On Test

1. **Visual inspection**:
   - No shorts visible
   - All connections secure
   - No damaged components

2. **Power test** (no firmware yet):
   ```
   Connect USB power
   Check: Red power LED on ESP32-CAM should light
   Measure: 3.3V between 3V3 and GND pins
   Measure: 5V between 5V and GND pins
   ```

3. **Firmware flash and serial test**:
   ```
   Flash firmware via ESP32-CAM-MB
   Open serial monitor (115200 baud)
   Check for:
     - Boot messages
     - WiFi connection
     - I2S initialization
     - Microphone test results
   ```

4. **Audio stream test**:
   ```bash
   # Download audio sample
   timeout 10 curl http://<ESP32_IP>:8080/stream > test.wav

   # Verify audio quality
   ffplay test.wav  # Should hear room ambient sound
   ```

### Microphone Validation

**Good audio characteristics:**
- ✅ Non-zero samples: >90%
- ✅ Dynamic range: -5000 to +5000 or wider
- ✅ Responds to sound (speak, clap, play music)
- ✅ Low noise floor when silent

**Problem indicators:**
- ❌ >99% zeros → wiring issue or pin swap needed
- ❌ Constant value (e.g., -30935, 0, -30935) → I2S protocol mismatch
- ❌ Very narrow range (-10 to +10) → gain too low
- ❌ Clipping (constant ±32767) → gain too high or mic damaged

## 📐 Mechanical Specifications

### ESP32-CAM Dimensions
- Length: 40.5mm
- Width: 27mm
- Height: 4.5mm (without camera)
- Mounting holes: None (use adhesive or case)

### INMP441 Dimensions
- Length: 15mm
- Width: 13mm
- Height: 3mm (module + mic)
- Mounting holes: 2 × 2mm (on some breakouts)

### Enclosure Requirements
- Minimum internal: 50 × 35 × 15mm
- Microphone clearance: 5mm (prevent muffling)
- Ventilation: Required for cooling and acoustic path

## 🔧 Maintenance

### Regular Checks (Monthly)

1. **WiFi signal strength**:
   ```bash
   curl http://<ESP32_IP>:8080/status | jq '.wifiRSSI'
   # Should be better than -70 dBm
   ```

2. **Audio quality**:
   ```bash
   timeout 10 curl http://<ESP32_IP>:8080/stream > test.wav
   ffplay test.wav  # Listen for clarity
   ```

3. **Uptime/stability**:
   ```bash
   curl http://<ESP32_IP>:8080/status | jq '.uptime'
   # Long uptime = stable operation
   ```

### Cleaning (Quarterly)

- **Dust removal**: Compressed air on microphone port
- **Contact cleaning**: Isopropyl alcohol on connectors (power off first)
- **Inspection**: Check for corrosion, loose wires, damaged insulation

### Replacement Schedule

- **Microphone**: 2-5 years (or if damaged)
- **ESP32-CAM**: 5+ years (solid-state, no wear parts)
- **Capacitors**: 5-10 years (electrolytic age)

## 🛒 Bill of Materials (BOM)

### Core Components

| Item | Qty | Unit Price | Total | Source |
|------|-----|------------|-------|--------|
| ESP32-CAM | 1 | $8 | $8 | Amazon/AliExpress |
| ESP32-CAM-MB | 1 | $4 | $4 | Amazon/AliExpress |
| INMP441 Microphone | 1 | $4 | $4 | Amazon/SparkFun |
| USB-A to USB-C cable | 1 | $3 | $3 | Amazon |
| Hookup wire (6 colors) | 1m | $5 | $5 | Amazon/eBay |

**Core Total: ~$24**

### Optional Components

| Item | Qty | Unit Price | Total | Purpose |
|------|-----|------------|-------|---------|
| 100µF capacitor | 1 | $0.50 | $0.50 | Power stability |
| 0.1µF capacitor | 1 | $0.10 | $0.10 | Noise filtering |
| Project enclosure | 1 | $5 | $5 | Protection |
| External WiFi antenna | 1 | $8 | $8 | Better range |
| USB power adapter (5V 2A) | 1 | $6 | $6 | Standalone power |

**Optional Total: ~$20**

**Grand Total: ~$44** (core + all optional)

### Tools Required (One-time)

- Soldering iron: $20-50
- Solder (lead-free): $10
- Wire strippers: $8
- Multimeter: $15
- Helping hands: $10

**Tool Total: ~$63-83** (if you don't already have)

## 📚 References

- [ESP32-CAM Schematic](https://github.com/SeeedDocument/forum_doc/raw/master/reg/ESP32_CAM_V1.6.pdf)
- [INMP441 Datasheet](https://invensense.tdk.com/wp-content/uploads/2015/02/INMP441.pdf)
- [I2S Protocol Specification](https://www.sparkfun.com/datasheets/BreakoutBoards/I2SBUS.pdf)
- [ESP32 I2S Driver Documentation](https://docs.espressif.com/projects/esp-idf/en/latest/esp32/api-reference/peripherals/i2s.html)

---

**Questions or issues? Check the [README.md](README.md) troubleshooting section or open a GitHub issue.**
