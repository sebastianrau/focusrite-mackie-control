# TouchOSC demo layout

`sweetspot.tosc` is a layout for [TouchOSC](https://hexler.net/touchosc)
with the volume fader, a stereo level meter with a POST switch (meter after
the volume), the speaker buttons with their names, Mute and Dim, and a Scene B switch (lit on scene B). Buttons and fader follow the app, so changes made
in the app, on the interface or on a control surface show up in TouchOSC too.
Speakers disabled in the app are shown grey and cannot be pressed (a small
script in the layout follows `/monitor/speaker/<x>/enabled`).

## Setup

**TouchOSC on the same Mac**

1. In TouchOSC, open `sweetspot.tosc`.
2. *Connections → OSC → Connection 1*: UDP, host `127.0.0.1`,
   send port `9000`, receive port `9001`.
3. In Sweetspot, *Configuration → OSC*: add the target
   `127.0.0.1:9001` and press *Save*.

**TouchOSC on a phone or tablet**

1. Copy `sweetspot.tosc` to the device and open it in TouchOSC.
2. *Connections → OSC → Connection 1*: UDP, host = the Mac's IP address,
   send port `9000`, receive port `9001`.
3. In Sweetspot, *Configuration → OSC*: tick *Allow other devices*,
   add the target `<device IP>:9001` and press *Save*.

Start the layout (play button). Fader, buttons, names and meter fill in within
a few seconds at the latest.

The addresses are documented in [docs/osc.md](../../docs/osc.md).

## Regenerating

The layout is generated from the app's address space:

```sh
go run ./cmd/touchosc-layout examples/touchosc/sweetspot.tosc
```
