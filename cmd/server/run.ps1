go build .
while ($true) 
{
  start-process -filepath .\server.exe -argumentlist "-loglevel debug -store " -wait -nonewwindow
}
