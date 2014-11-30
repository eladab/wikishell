#include <stdio.h>
#include <sys/types.h>
#include <fcntl.h>
#include <unistd.h>
#include <termios.h>
#include <signal.h>

struct termios old_tio;

void rawmodeon()
{
	struct termios tio;
	tcgetattr(0, &old_tio);
	tio = old_tio;	
	tio.c_lflag &= (~ICANON & ~ECHO);	
	tcsetattr(0, TCSANOW, &tio);
}

void rawmodeoff() 
{
	tcsetattr(0, TCSANOW, &old_tio);
}
