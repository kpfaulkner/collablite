# TODO

- Client connecting will get ALL properties for the object as part of the normal update stream.
  Might be easier that having some special case for the first update.
- Graphical client for demo purposes.
- Unit tests
- Documentation
- Postgres(?) support in addition to Sqlite
- ~~Currently client wont get receive updates until it does one itself. Allow it to just register for a stream of updates.~~
  Done. Just make sure client sends empty change with the appropriate objectid