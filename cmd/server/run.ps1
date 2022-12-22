go build .
while ($true) 
{
  start-process -filepath .\server.exe -argumentlist "-loglevel debug " -wait -nonewwindow
}
