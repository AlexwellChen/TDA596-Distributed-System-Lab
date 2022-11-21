1. Use `docker build -t [name]:[version] .` inside server folder to build a docker image. Note the `.` (dot) represents the current path
(Alternative: use `docker build -t server:multistage -f Dockerfile.multistage .` to create a multistage image)
(multistage image is smaller in size (around 30mb insteand of normal ones around 400mb. But with no terminal option.)

2. Run `docker run -p 8080:8080 [name]:[version]` to start a container

Note docker uses an isolated network from host machine, 8080:8080 maps the container port to host port. The format of the --publish command is [host_port]:[container_port]. So if we wanted to expose port 8080 inside the container to port 3000 outside the container, we would pass 3000:8080 to the --publish flag.

Dockerfile uses parameter `8080` while running server, so the container_port in our case should always be 8080

3. Run `docker ps`
    `docker inspect [container_name] | grep IP` to check the ip address of docker container

4. After running client, use `container_ip:8080` instead of localhost
    