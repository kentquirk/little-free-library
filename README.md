# Little Free Library
Provides the support for the Digital Little Free Library project, which is a frontend for Project Gutenberg.

More to come as I develop this.

## Design Notes

### pkg/books is too big

We have a rather odd data structure to draw from -- the Project Gutenberg database. It's an XML file format that has evolved over the years -- it's now a collection of individual files, each containing information about a single book. There's a single directory with over 65000 subdirectories, each containing an XML file. That's a lot of parsing and it takes over a minute to load all that data.

Furthermore, although the data follows specific XML schema (called RDF), many times the schema named do not themselves exist! And in other cases, the schema are layered deep within some other specification. So I've had to do a bit of guessing to determine the meaning of all the fields.

The original hope was to cleanly separate the XML file reading code from the book data code -- but it proved challenging to keep them separated (the code to translate one to the other gets tricky), so I eventually gave up and moved the RDF loader/parser into the more general books package. Maybe someday we can move it back out.

The other thing I would have liked to do is to separate the data storage and make it more generic, but (see below) I also wanted this whole thing to run in memory and not have an external dependency for now. So that also ends up in the books package for now.

### Technical details

* Using a lightweight framework in Go makes things like TLS integration a little easier. I had heard good things about chi, but the developers there have decided that they don't like Go's versioning system and have deliberately broken it (going backwards in version numbers from 4.X to 1.5!) and then [doubled down on justifying their decision](https://github.com/go-chi/chi/issues/561). You don't have to like the way Go did it (I don't, although I don't hate it the way I did at first). But deliberately fighting with the platform standard is pointless and makes your project unusable in production. So I've settled on [echo](https://echo.labstack.com/), which seems to be stable, updated, lightweight, and reasonably popular.
* Echo has a bunch of useful middleware that will be incorporated:
    * Recover, for dealing with errors.
    * Authentication. We want the ability to issue developer keys that we can throttle and/or turn off to limit abuse.
    * Logger, for tracking performance etc.
    * RequestID, so that we can do tracing of requests in our diagnostics
    * Static can easily serve static files when/if we start delivering web pages instead of just data.
    * BodyLimit allows the server to limit the size of a request body (another form of abuse) - although we may not need it since so far we only have GET requests.
* Echo also supports graceful shutdown, which is a nice thing to have, so we'll set that up.
* Echo supports TLS through Let's Encrypt, so we'll enable that if the port is specified as 443. However, for our deployment we may just deploy on port 80 and leave the SSL termination to the load balancer (AWS supports that easily).
* We're using [a config library](https://github.com/codingconcepts/env) to avoid individually handling a bunch of environment variables.
* Data storage: we want a lightweight storage system that is easy to use; we don't need a lot. Options are:
    * Redis is fast, easy, powerful and rock-solid. But it requires setting it up and a more complex deployment. The same applies to third parties like, say, Atlas.
    * SQLite can be used with a local setup on a local disk.
    * One of the AWS data storage solutions; none of them are easy and they can quickly rack up the bills.
    * AirTable might be a really good solution -- we let data entry happen in AirTable, and refresh the data on startup or when prompted by the API.
        * It would only be used to store basic information on a curated list of titles.
        * Actual content would be fetched and cached from project gutenberg.
        * The free version would be limited to 1200 items, which might be fine for a little library but it is definitely limiting the content and requires a fair bit of initial curation.
        * Should abstract it a bit so that it's easy to implement other backends.
    * For now, since this is a read-only API, we're going to store all the data in a local cache and reload it from the source data every time we start the server. This costs about a minute at startup but avoids any of these problems. However, it also causes problems if we ever want to scale horizontally and also prevents us from running this service on Lambda (there's no persistence in Lambda). We can revisit this later once we have a better understanding of our query types.


### Reading the data from Project Gutenberg

[This package](pkg/rdf) contains the code to load an "RDF" file which is a specific format of XML that is used by Project Gutenberg.
[The offline catalogs page](http://www.gutenberg.org/ebooks/offline_catalogs.html) requests that this format be the one used for fetching data for offline uses.


