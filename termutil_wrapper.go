package main

/*
#include "termutil.h"
*/
import "C"

func SetRawModeEnabled(enabled bool) {
	if enabled {
		C.rawmodeon()
	} else {
		C.rawmodeoff()
	}
}
