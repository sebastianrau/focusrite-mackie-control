# focusrite-mackie-control  <img src="logo.png" width="15%" height="15%"> 

This project enables remote control of Focusrite devices via a Mackie Control surface.

## Requirements

- Go 1.23.5 or later
- A compatible Focusrite device
- A Mackie Control-compatible control surface
- [Fyne](https://fyne.io/) as GUI framework

## Installation

1. Clone the repository:
   
   ```bash
   git clone https://github.com/sebastianrau/focusrite-mackie-control.git
   cd focusrite-mackie-control
   ```

2. Install dependencies:
   
   ```bash
   go mod tidy
   ```

3. Build the application:
   
   ```bash
   make
   ```

## Usage

Start the application with:

```bash
./build/bin/focusrite-mackie-control
```

The application allows you to control your Focusrite device via your Mackie Control surface.

## Supported Devices

This project is designed for use with Focusrite devices. Tested with:
- Focusrite Scarlett 4i4 3rd Gen
- Focusrite Scarlett 18i20 3rd Gen

## License

Due to 
This project is licensed under the MIT License.
