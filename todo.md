# refactoring

[x] Put Date into own package
[x] Move book types into own package
[ ] Give types the ability to marshal to JSON
[ ] PGFile splits out compression and format
[ ] Make a separate data structure for Files (not an array of PGFile, use map[format]pgindex)
[ ] Create general-purpose database package that accepts queries and returns book types
[ ] Add indexes for bookid, format
[ ] database also needs update actions
[ ] build a backend for mongo
[x] Move RDF into its own package
[ ] write a loader that reads RDF into a data structure
[ ] write a saver that writes entries to database


# project general
[x] upgrade to go 1.16
[ ] start using goreleaser
[ ] start using [go:embed](https://golang.org/pkg/embed/) for templates and static files
[ ] add throttling per user
[ ] add query endpoint to select from a random collection of "interesting" queries
[ ] Improve date testing

# deployment
[ ] consider building a docker manifest for mongo and the app
[ ] figure out how to do this with lambda once mongo exists

