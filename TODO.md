# TODO

- Client connecting will get ALL properties for the object as part of the normal update stream.
  Might be easier that having some special case for the first update.
- Unit tests
- Documentation
- Postgres(?) support in addition to Sqlite
- Switch to sqlx
- Channel filling up and blocking all clients is a concern. Believe it is fixed but need tests and more thought.
- Add Object <--> JSON converters
- ~~Graphical client for demo purposes.~~
- ~~Sanitise messages from clients (SQL injection etc) (prepared statements)~~
- ~~Make proper client lib that has callbacks as opposed to making the caller use channels directly~~
- ~~Currently client wont get receive updates until it does one itself. Allow it to just register for a stream of updates.~~
  Done. Just make sure client sends empty change with the appropriate objectid


