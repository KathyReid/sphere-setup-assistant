# sphere-setup-assistant

sphere-setup-assistant - help ensure that the sphere is always connected to some kind of network

# REBOOT AND RESET SUPPORT

Sphere setup assistant monitors the hardware reset button and then initiates one of 3 different kinds
of reset action depending on the length of the press.

* short press - 3 seconds or less - initiates a system reboot
* longer press - 3 - 6 seconds - initiates a user data reset
* very long press - 6+ seconds - initiates a factory reset

Currently the user data reset is implemented with sphere-reset --reset-setup although this may change in future. Currently the factory reset function does the same thing as the user data reset. This will change in the future.

When the reset button is first pressed, the led matrix shows solid green, indicating a reboot will occur. If the button is held longer, the led matrix shows solid yellow indicating a user data reset will occur. Finally, if the button is held even longer,
the led matrix show a sold red indicating a factory reset will occur.

When the button is released, the color corresponding to the selected mode fades until the action occurs.

# License

Copyright (c) 2015 Ninjablocks Inc licensed under the MIT license
