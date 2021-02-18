# refactoring

* Move book types into own package
    * Give them the ability to marshal to JSON
    * PGFIle splits out compression and format
    * Make a separate data structure for Files (not an array of PGFile, use map[format]pgindex)
    * Add indexes for bookid, format
* Create general-purpose database package that accepts queries and returns book types
    * database also needs update actions
    * build a backend for mongo
* Move RDF into its own package
    * write a loader that reads RDF into a data structure
    * write a saver that writes entries to database


# project general
* upgrade to go 1.16
* start using goreleaser
* start using [embed](https://golang.org/pkg/embed/) for templates and static files
* add throttling per user
* add query endpoint to select from a random collection of "interesting" queries

# deployment
* consider building a docker manifest for mongo and the app
* figure out how to do this with lambda once mongo exists

