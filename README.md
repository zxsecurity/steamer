![Steamer Logo](https://i.imgur.com/FobEdvM.png)

Steamer
=======

Import, manage, search public dumps.

Do you have massive amounts of CSV, .sql, .txt, that have credentials, passwords, and hashes inside?
Use Steamer to manage them! Load them into a MongoDB database, and either use the console directly, or just use the handy web interface (complete with JSON export).

Install
-------

- Install Go and MongoDB.
- `go get gopkg.in/mgo.v2 && go get github.com/gorilla/mux`

At this point, it is recommended to import one of the more simple breaches that do not require an index to import.

- `go run ./importers/adobe.go`

Now we need to create relevant indexes for MongoDB:
- In the mongo console, create indexes as:
  - memberid: hashed
  - breach: 1
  - email: 1
  - liame: 1
  - passwordhash: 1
  
The commnds to create the indexes are:
- `mongo`
- `use steamer`
- `db.dumps.createIndex( { memberid: "hashed"}, { background: true} )`
- `db.dumps.createIndex( { breach: 1}, { background: true} )`
- `db.dumps.createIndex( { email: 1}, { background: true} )`
- `db.dumps.createIndex( { liame: 1}, { background: true} )`
- `db.dumps.createIndex( { passwordhash: 1}, { background: true} )`

Install complete!

Running Steamer
---------------

If you're smart, you'll consider running nginx in front of go, but we're lazy, so really just run:
`go run ./steamer.go`.

Write an importer
-----------------

Copy the `importers/importer-template.go` file as appropriate. Fill it in with relevant code. See the other importers for examples.
That template is threaded and designed for CSVs. See `./importers/linkedin2016.go` for a more complex example.

If you write an importer for a public breach, please send a pull request so everyone can import it too. Please note that no public breaches are provided here in the repository itself.

Problems?
---------

Make sure you're running MongoDB 3.0 or higher. Previous versions have had issues with indexes not working properly, and there is some new syntax which requires this version.

Performance? Try tweaking your MongoDB configuration file to turn off journaling and enabling the new database engine.
