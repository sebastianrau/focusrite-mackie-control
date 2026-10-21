# focusrite-mackie-control <img src="logo.png" width="15%" height="15%" align="right">

A monitor controller for Focusrite interfaces. It switches between up to four
speaker pairs and a subwoofer, sets the monitor volume, and handles mute and
dim. You can operate it from:

- its own window and the menu bar / system tray,
- a Mackie Control (MCU) compatible surface,
- a Stream Deck with a MIDI plugin (profile and icons in [`streamdeck/`](streamdeck/)),
- the monitor mute and dim buttons on the interface itself.

Every control stays in sync with the others: pressing Dim on the interface
lights the Dim key on the Stream Deck and in the window.

## Requirements

- A Focusrite Scarlett interface and **Focusrite Control** running on the same computer.
  The app talks to the interface through Focusrite Control's remote protocol.
- Optional: a Mackie Control compatible surface or a Stream Deck with a MIDI plugin.

Tested with the Scarlett 4i4 (3rd Gen) and Scarlett 18i20 (3rd Gen).

## Installation

Download the archive for your system from the
[latest release](https://github.com/sebastianrau/focusrite-mackie-control/releases/latest),
unpack it and start **Monitor Controller**.

The macOS app is not signed. On first start, right-click it and choose **Open**.

## First start

1. Start Focusrite Control and connect the interface.
2. Start Monitor Controller. Focusrite Control asks you to allow the new remote
   device **Monitor Controller**. Allow it.
3. With no interface configured yet, the app takes the first one it finds and assigns its
   stereo outputs in order: outputs 1-2 to Speaker A, 3-4 to B, 5-6 to C, 7-8 to D,
   9-10 to the sub. Change this in the **Configuration** window (tray menu), section *Focusrite*.

The app controls the gain, mute and name of the assigned outputs.

## Speakers, volume, mute and dim

- **Speakers**: selecting an *exclusive* speaker turns the other speakers off.
  The sub is switched on its own. Unused speakers can be disabled in the
  **Configuration** window, section *Monitor Controller*.
- **Volume** is applied to all enabled outputs, from -127 dB to 0 dB.
- **Dim** lowers the volume by an adjustable amount (20 dB by default).
- **Mute** mutes all outputs.

## Control surface and Stream Deck

Choose the MIDI input and output ports in the **Configuration** window, section *Midi*. With a
Stream Deck, create two virtual MIDI ports (on macOS: IAC Driver buses in
*Audio MIDI Setup*), point the Stream Deck MIDI plugin at them, and import the
profile and icon pack from [`streamdeck/`](streamdeck/).

Default mapping (matching the Stream Deck profile):

| Function     | MCU control       |
| ------------ | ----------------- |
| Volume       | Fader 1           |
| Mute         | Mute 1            |
| Dim          | Solo 1            |
| Speaker A–D  | Trim, Touch, Write, F4 |
| Sub          | Read              |

Every assignment can be changed in the Configuration window. Pressing the select button of
the volume fader moves the fader to the current volume.

## Settings file and logs

Settings are stored in a YAML file in the user configuration folder:

| System  | Path |
| ------- | ---- |
| macOS   | `~/Library/Application Support/Monitor-Controller/monitor-controller.yaml` |
| Windows | `%AppData%\Monitor-Controller\monitor-controller.yaml` |
| Linux   | `~/.config/Monitor-Controller/monitor-controller.yaml` |

Files from 0.1.x are converted to the new format on first start. A file that
cannot be read is renamed to `monitor-controller.yaml.broken`, and the app
starts with default settings.

The app logs to stderr. Set `MONITOR_CONTROL_LOG=debug` to see more detail.
To try changes without touching your settings, run with `-config <file>`.

## Development

Requires Go 1.25+ and a C compiler (Fyne and the MIDI driver use cgo). On
Linux, also install `libgl1-mesa-dev xorg-dev libasound2-dev`.

```sh
make run      # run from source
make test     # all tests with the race detector
make lint     # golangci-lint
make package  # build a release archive for this OS into dist/
make help     # all targets
```

Pushing a tag builds the release archives for macOS (arm64 and amd64),
Windows and Linux, and publishes them as a GitHub release.

### Architecture

```
  Focusrite Control ── focusrite.Client ── focusrite.Adapter ─┐
                                                              │ commands
  MCU / Stream Deck ── mcu.MidiSurface ─── mcu.Adapter ───────┼──────────► controller ──► config.Store
                                                              │ ◄──────────     │
  Window and tray ──────────────────────── ui.App ────────────┘  snapshots      │
                                                                                └─ domain (state and rules)
```

- `internal/domain` holds the state and every rule as pure functions
  (exclusive speakers, dim, volume limits, which speaker drives the meter).
- `internal/controller` applies commands one at a time on a single goroutine.
  It publishes each new state as a value snapshot. A subscriber that falls
  behind gets only the latest snapshot and never blocks the others.
- The adapters translate between snapshots and devices. They send only what
  changed, and turn device input into commands.
- `internal/focusrite/protocol` is the XML protocol of Focusrite Control. The
  test data contains real device messages from the 18i20 and 4i4.

## License

See [LICENSE.md](LICENSE.md).
