# memebase
Codebase for a JSON RESTful API to store and retrieve base64 encoded images(memes)

Endpoints and Routing
![image](https://github.com/michaelgov-ctrl/memebase/assets/81777732/6378de24-956f-4e93-b004-c06312c081cc)

example post:
curl --header "Content-Type: application/json" \
  --request POST \
  --data '{"artist":"test","title":"test","b64":"test"}' \
  http://localhost:4000/v1/memes
