You can use docker and docker-compose to spin up a local instance of stopwatch for development!

NOTE: Please do **NOT** run these docker images in any production setting. The vault has been initialized already so that the application works with predetermined seal keys.

`docker-compose up` should work

It uses the source code in the parent directory to compile a new binary and run it when the container starts. This is for development purposes only, to get people up and running quickly. 
