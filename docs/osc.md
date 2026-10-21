# OSC interface

Monitor Controller runs an OSC server, so any OSC client can operate it and
show its state: TouchOSC, Open Stage Control, a DAW, a script. A ready-made
TouchOSC layout is in [`examples/touchosc/`](../examples/touchosc/).

## Connection

| Setting | Default | Configuration window, section *OSC* |
| ------- | ------- | ----------------------------------- |
| Server on/off | on | *OSC Server* |
| Port | UDP 9000 | *Port* |
| Reachable from | this computer only | *Network: Allow other devices* |
| Targets | none | *Targets*, one `host:port` per line |

OSC has no authentication. Anyone who can reach the port can control your
monitors, so allow other devices only on networks you trust. Targets on
other devices also require *Allow other devices*.

The app sends updates to:

- **targets** from the configuration, always;
- **subscribers**: a client sends `/monitor/subscribe` (optionally with the
  port it receives on as an int argument; the default is the port it sent
  from) and repeats it at least every 60 seconds. `/monitor/unsubscribe`
  ends it early.

Changes are sent as they happen. The complete state is also repeated every
5 seconds, since UDP can drop packets. `/monitor/sync` sends it at once to
all recipients and to the sender.

## Addresses

Every state address works both ways: send it to change the value, receive it
to see the current value. Numbers may be sent as int, float or bool.

| Address | Value | Direction |
| ------- | ----- | --------- |
| `/monitor/volume` | float, dB −127…0 | in / out |
| `/monitor/volume/normalized` | float 0…1, linear over −127…0 dB | in / out |
| `/monitor/mute` | int 0/1 | in / out |
| `/monitor/mute/toggle` | – | in |
| `/monitor/dim` | int 0/1 | in / out |
| `/monitor/dim/toggle` | – | in |
| `/monitor/speaker/{a,b,c,d,sub}` | int 0/1 | in / out |
| `/monitor/speaker/{a,b,c,d,sub}/toggle` | – | in |
| `/monitor/speaker/{a,b,c,d,sub}/name` | string | out |
| `/monitor/speaker/{a,b,c,d,sub}/enabled` | int 0/1 | out |
| `/monitor/meter/left`, `/monitor/meter/right` | float, dBFS | out, ≤ 20/s |
| `/monitor/meter/left/normalized`, `…/right/normalized` | float 0…1 over −60…0 dB | out, ≤ 20/s |
| `/monitor/device/connected` | int 0/1 | out |
| `/monitor/device/model` | string | out |
| `/monitor/sync` | – | in |
| `/monitor/subscribe`, `/monitor/unsubscribe` | [int port] | in |

Rules:

- **Toggles** (`…/toggle`) act on "on" values only, or on a message without
  arguments. A momentary button sends 1 on press and 0 on release, so the
  release is ignored.
- **Selecting a speaker** follows the app's rules: an exclusive speaker turns
  the other speakers off, and the app sends their new state. A disabled
  speaker ignores the request.
- **Meter** values are the levels Focusrite Control reports for the selected
  speaker's output, before the volume.

## Examples

With `oscsend` from [liblo](https://liblo.sourceforge.net):

```sh
oscsend localhost 9000 /monitor/volume f -30
oscsend localhost 9000 /monitor/speaker/b i 1
oscsend localhost 9000 /monitor/mute/toggle
oscsend localhost 9000 /monitor/subscribe i 9001   # then listen on UDP 9001
```
