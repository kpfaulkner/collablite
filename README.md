# Collablite

Conflict free (mostly) data sharing service. Inspired by the Figma [post](https://www.figma.com/blog/how-figmas-multiplayer-technology-works/)

aka, CRDT without the CRDT bit :)

## What is it?

Collablite is a service that allows you to share data between multiple clients. It is inspired by the Figma post on their multiplayer technology.
It is not a CRDT implementation, but it does use a similar approach to allow multiple clients to share data without conflict.

## How does it work?

There are a number of key features/conditions that this service provides:

- For a given object being edited (by multiple clients) the object exists ONLY in a single instance of the service. This may
  seem like a scaling issue in the future, but given that it's NOT expected that a LOT of changes will be happening to a single
  document at any one time, this should be safe. IF the instance of the service dies, then a new one can be fired up immediately
  and all clients can reconnect and continue. The state of the object at the time the service died is persisted so very little (if any)
  changes should be lost. Currently this is deemed acceptable.

  If the situation arises where a single instance of the service (for a specific object) is NOT sufficient and horizontal scaling would
  be required to meet the load, then a solution would be investigated then, but I don't want to go down that route yet.

- If more then one instance is required (to handle the general load, NOT specifically for one object) then the load balancer
  mechanism used will need to have some support for server affinity. If affinity cannot be handled then changes will NOT be
  shared correctly across clients.

- The resolution of concurrent conflicts of an object is that "last write wins". This is a simple approach but works well.
  Please see the Figma [post](https://www.figma.com/blog/how-figmas-multiplayer-technology-works/) for more details.


## Technologies used

TODO

## Architecture Diagram

TODO

## How to use it

TODO
