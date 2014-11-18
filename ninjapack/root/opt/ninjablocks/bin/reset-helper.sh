#!/bin/sh

main()
{
	cmd=$1
	case "$1" in
	init-io)
		if test -e /sys/kernel/debug/omap_mux/xdma_event_intr1; then
			# in later kernels (3.12) this path won't exist
			echo 37 > /sys/kernel/debug/omap_mux/xdma_event_intr1
			echo 20 > /sys/class/gpio/export
			echo in > /sys/class/gpio/gpio20/direction
		fi

		# to read the reset button
		# cat /sys/class/gpio/gpio20/value
	;;
	reboot)
		/sbin/reboot
	;;
	reset-userdata)
		# TBD: write scripts that will reset the user-data
		sphere-reset --reset-setup
		/sbin/reboot
	;;
	reset-root)
		# TBD: write scripts that will reset the root partition
		sphere-reset --reset-setup
		/sbin/reboot
	;;
	esac

}


main "$@"
