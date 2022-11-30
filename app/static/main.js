const BLOCK_SIZE = 20;
const WALL_SIZE = 2;
var colorMap;
var wait;
var intervalId;

async function init() {
  await fetchColorMap();
  setInterval(fetchWait, 1000);
}

async function fetchWait() {
  var newWait;
  await fetch('/wait')
    .then((response) => response.json())
    .then((json) => {
            newWait = json;
    });
  if (newWait != wait) {
    console.log("newWait detected", "wait:", wait, "newWait:", newWait)
    if (intervalId !== undefined) {
      console.log("clearing existing interval")
      clearInterval(intervalId);
    }
    intervalId = setInterval(fetchBoard, newWait);
    wait = newWait;
  }
}

async function fetchColorMap() {
        console.log("fetching colorMap")
        return fetch('/colors')
                .then((response) => response.json())
                .then((json) => {
                        colorMap = new Map(json);
                });
}

async function fetchBoard() {
        return fetch('/board')
                .then((response) => response.json())
                .then((board) => {
                        draw(board);
                });
}

async function move(op) {
  const data = { "op": op };
  console.log(data);
  const param = {
    method: "POST",
    headers: {
      "Content-Type": "application/json; charset=utf-8"
    },
    body: JSON.stringify(data)
  };

  return fetch('/actions', param)
    .then((response) => {
      if (response.ok) {
        fetchBoard();
      } 
    });
}

document.addEventListener('keydown', (event) => {
  switch (event.key) {
		case 'ArrowUp':
			move('rotate');
      event.preventDefault();
			break;
		case 'ArrowDown':
			move('down');
      event.preventDefault();
			break;
		case 'ArrowLeft':
			move('left');
      event.preventDefault();
			break;
		case 'ArrowRight':
			move('right');
      event.preventDefault();
			break;
    case ' ':
      move('drop');
      event.preventDefault();
      break;
	}
});

function newGame() {
  console.log("new game")
  const param = {
    method: "POST",
    headers: {
      "Content-Type": "application/json; charset=utf-8"
    }
  };
  return fetch('/board', param);
}

function draw(json) {
  var canvas = document.getElementById("stage");
  canvas.setAttribute("width", String(json.width * BLOCK_SIZE + WALL_SIZE*2));
  canvas.setAttribute("height", String(json.height * BLOCK_SIZE + WALL_SIZE));
  var ctx = canvas.getContext("2d");

  // draw the lattice
  ctx.beginPath();
  ctx.strokeStyle = "rgb(220, 220, 220)";
  ctx.lineWidth = 1;
  for (let i = 0; i < json.height + 1; i++) {
      ctx.moveTo(WALL_SIZE, i * BLOCK_SIZE);
      ctx.lineTo(WALL_SIZE + json.width * BLOCK_SIZE, i * BLOCK_SIZE);
  }
  for (let j = 0; j < json.width + 1; j++) {
      ctx.moveTo(WALL_SIZE + j * BLOCK_SIZE, 0);
      ctx.lineTo(WALL_SIZE + j * BLOCK_SIZE, json.height * BLOCK_SIZE);
  }
  ctx.stroke();
  ctx.closePath();

  // draw the frame
  ctx.beginPath();
  ctx.fillStyle = "rgb(100, 100, 100)";
  ctx.fillRect(0, 0, WALL_SIZE, json.height * BLOCK_SIZE + WALL_SIZE);
  ctx.fillRect(WALL_SIZE + json.width * BLOCK_SIZE, 0, WALL_SIZE, json.height * BLOCK_SIZE + WALL_SIZE);
  ctx.fillRect(0, json.height * BLOCK_SIZE, json.width * BLOCK_SIZE + WALL_SIZE*2, WALL_SIZE);
  ctx.stroke();
  ctx.closePath();

  // draw the board
  for (let i = 0; i < json.height; i++) {
    for (let j = 0; j < json.width; j++) {
      if (json.data[i][j] != 0) {
        color = colorMap.get(json.data[i][j])
        ctx.fillStyle = color;
        ctx.fillRect(WALL_SIZE + j * BLOCK_SIZE + 2, i * BLOCK_SIZE + 2, BLOCK_SIZE - 1, BLOCK_SIZE - 1);

        ctx.beginPath();
        ctx.strokeStyle = "rgb(64, 64, 64)";
        ctx.lineWidth = 1;
        ctx.rect(WALL_SIZE + j * BLOCK_SIZE, i * BLOCK_SIZE, BLOCK_SIZE, BLOCK_SIZE);
        ctx.stroke();
        ctx.closePath();
      }
    }
  }
}

init();
