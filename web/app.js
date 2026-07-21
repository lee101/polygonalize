const $ = (query) => document.querySelector(query);
const canvas = $('#output');
const ctx = canvas.getContext('2d', { alpha: true });
const source = document.createElement('canvas');
const sourceCtx = source.getContext('2d', { willReadFrequently: true });
const video = $('#video');

let mesh = null;
let seed = 17;
let session = null;
let raf = 0;
let lastFrame = 0;
let fileBase = 'polygonalized';
let view = 'flat';
let three = null;
let rerunTimer = 0;
let activeMediaURL = '';
let customPrimitive = null;
let customPrimitiveURL = '';
let customPrimitiveData = '';

const opts = () => ({
  triangles: +$('#triangles').value,
  edgeBias: +$('#edgeBias').value / 100,
  stability: +$('#stability').value / 100,
  seed,
});

const primitive = () => $('#primitive').value;
const fit = (width, height, maxSize = 1100) => {
  const scale = Math.min(1, maxSize / Math.max(width, height));
  return [Math.max(2, Math.round(width * scale)), Math.max(2, Math.round(height * scale))];
};

function imageData() {
  return sourceCtx.getImageData(0, 0, source.width, source.height);
}

function runFrame(stable = false) {
  if (!window.polygonalizeReady || !source.width) return;
  const data = imageData();
  const json = stable
    ? window.polygonalizeFrame(session, data.data, data.width, data.height, JSON.stringify(opts()))
    : window.polygonalizeImage(data.data, data.width, data.height, JSON.stringify(opts()));
  mesh = JSON.parse(json);
  draw();
  $('#stats').textContent = `${mesh.triangles.length.toLocaleString()} triangles`;
}

function triangleMetrics(triangle) {
  const a = mesh.points[triangle.a];
  const b = mesh.points[triangle.b];
  const c = mesh.points[triangle.c];
  const area = Math.abs((b.x - a.x) * (c.y - a.y) - (b.y - a.y) * (c.x - a.x)) / 2;
  return { a, b, c, area: Math.max(0.01, area), x: (a.x + b.x + c.x) / 3, y: (a.y + b.y + c.y) / 3 };
}

function regularPoints(x, y, radius, sides, rotation = 0) {
  return Array.from({ length: sides }, (_, index) => {
    const angle = rotation + index * Math.PI * 2 / sides;
    return { x: x + Math.cos(angle) * radius, y: y + Math.sin(angle) * radius };
  });
}

function primitivePoints(triangle, kind = primitive()) {
  const m = triangleMetrics(triangle);
  const side = Math.sqrt(m.area) * 0.96;
  switch (kind) {
    case 'circle': return regularPoints(m.x, m.y, Math.sqrt(m.area / Math.PI) * 0.96, 18, -Math.PI / 2);
    case 'square': {
      const half = side / 2;
      return [{ x: m.x - half, y: m.y - half }, { x: m.x + half, y: m.y - half }, { x: m.x + half, y: m.y + half }, { x: m.x - half, y: m.y + half }];
    }
    case 'diamond': {
      const radius = side / Math.SQRT2;
      return [{ x: m.x, y: m.y - radius }, { x: m.x + radius, y: m.y }, { x: m.x, y: m.y + radius }, { x: m.x - radius, y: m.y }];
    }
    case 'hexagon': return regularPoints(m.x, m.y, Math.sqrt(2 * m.area / (3 * Math.sqrt(3))) * 0.96, 6);
    default: return [m.a, m.b, m.c];
  }
}

function tracePoints(points) {
  ctx.beginPath();
  ctx.moveTo(points[0].x, points[0].y);
  for (let index = 1; index < points.length; index++) ctx.lineTo(points[index].x, points[index].y);
  ctx.closePath();
}

function draw() {
  if (!mesh) return;
  if (view === 'three') {
    showThree();
    return;
  }
  $('#threeStage').hidden = true;
  canvas.hidden = false;
  canvas.width = mesh.width;
  canvas.height = mesh.height;
  ctx.fillStyle = '#111820';
  ctx.fillRect(0, 0, canvas.width, canvas.height);
  const kind = primitive();
  for (const triangle of mesh.triangles) {
    if (kind === 'custom' && customPrimitive) {
      const m = triangleMetrics(triangle);
      const size = Math.sqrt(m.area) * 1.38;
      ctx.drawImage(customPrimitive, m.x - size / 2, m.y - size / 2, size, size);
      continue;
    }
    tracePoints(primitivePoints(triangle, kind === 'custom' ? 'triangle' : kind));
    ctx.fillStyle = triangle.color;
    ctx.fill();
    if (kind === 'triangle' || view === 'wire') {
      ctx.strokeStyle = view === 'wire' ? 'rgba(240,244,255,.34)' : triangle.color;
      ctx.lineWidth = view === 'wire' ? 0.7 : 1;
      ctx.stroke();
    }
  }
}

async function showThree() {
  canvas.hidden = true;
  const host = $('#threeStage');
  host.hidden = false;
  $('#renderer').textContent = 'Three.js mesh preview';
  if (!three) {
    const THREE = await import('https://cdn.jsdelivr.net/npm/three@0.179.1/build/three.module.js');
    const renderer = new THREE.WebGLRenderer({ antialias: true });
    renderer.setPixelRatio(Math.min(devicePixelRatio, 2));
    renderer.setClearColor(0x10151c);
    host.replaceChildren(renderer.domElement);
    const scene = new THREE.Scene();
    const camera = new THREE.PerspectiveCamera(40, 1, 0.1, 100);
    camera.position.set(0, 0, 3.1);
    scene.add(new THREE.AmbientLight(0xffffff, 1.5));
    three = { THREE, renderer, scene, camera, object: null };
    host.addEventListener('pointermove', (event) => {
      const rect = host.getBoundingClientRect();
      camera.position.x = ((event.clientX - rect.left) / rect.width - 0.5) * 0.7;
      camera.position.y = -((event.clientY - rect.top) / rect.height - 0.5) * 0.45;
      camera.lookAt(0, 0, 0);
    });
    new ResizeObserver(() => {
      const rect = host.getBoundingClientRect();
      renderer.setSize(rect.width, rect.height, false);
      camera.aspect = rect.width / rect.height;
      camera.updateProjectionMatrix();
    }).observe(host);
    const loop = () => {
      if (three) {
        three.renderer.render(three.scene, three.camera);
        requestAnimationFrame(loop);
      }
    };
    loop();
  }
  const { THREE } = three;
  if (three.object) {
    three.scene.remove(three.object);
    three.object.geometry.dispose();
    if (three.object.material.map) three.object.material.map.dispose();
    three.object.material.dispose();
  }
  const positions = [];
  const colors = [];
  const uvs = [];
  const color = new THREE.Color();
  const pushVertex = (point, triangleColor, uv = null) => {
    const x = point.x / mesh.width - 0.5;
    const y = 0.5 - point.y / mesh.height;
    positions.push(x * 2, y * 2, Math.sin(x * 5) * 0.035 + Math.cos(y * 6) * 0.035);
    color.set(triangleColor);
    colors.push(color.r, color.g, color.b);
    if (uv) uvs.push(uv[0], uv[1]);
  };
  const kind = primitive();
  for (const triangle of mesh.triangles) {
    if (kind === 'custom' && customPrimitive) {
      const m = triangleMetrics(triangle);
      const half = Math.sqrt(m.area) * 0.69;
      const shape = [{ x: m.x - half, y: m.y - half }, { x: m.x + half, y: m.y - half }, { x: m.x + half, y: m.y + half }, { x: m.x - half, y: m.y + half }];
      for (const [index, uv] of [[0, [0, 1]], [1, [1, 1]], [2, [1, 0]], [0, [0, 1]], [2, [1, 0]], [3, [0, 0]]]) pushVertex(shape[index], '#ffffff', uv);
      continue;
    }
    const shape = primitivePoints(triangle, kind === 'custom' ? 'triangle' : kind);
    for (let index = 1; index + 1 < shape.length; index++) {
      pushVertex(shape[0], triangle.color);
      pushVertex(shape[index], triangle.color);
      pushVertex(shape[index + 1], triangle.color);
    }
  }
  const geometry = new THREE.BufferGeometry();
  geometry.setAttribute('position', new THREE.Float32BufferAttribute(positions, 3));
  geometry.setAttribute('color', new THREE.Float32BufferAttribute(colors, 3));
  if (uvs.length) geometry.setAttribute('uv', new THREE.Float32BufferAttribute(uvs, 2));
  geometry.computeVertexNormals();
  let material;
  if (kind === 'custom' && customPrimitive) {
    const texture = new THREE.Texture(customPrimitive);
    texture.colorSpace = THREE.SRGBColorSpace;
    texture.needsUpdate = true;
    material = new THREE.MeshBasicMaterial({ map: texture, transparent: true, alphaTest: 0.01, side: THREE.DoubleSide });
  } else {
    material = new THREE.MeshStandardMaterial({ vertexColors: true, roughness: 0.88, metalness: 0.04, side: THREE.DoubleSide });
  }
  three.object = new THREE.Mesh(geometry, material);
  three.scene.add(three.object);
}

function generatedDemo() {
  source.width = 960;
  source.height = 620;
  const gradient = sourceCtx.createLinearGradient(0, 0, 960, 620);
  gradient.addColorStop(0, '#7b52e6');
  gradient.addColorStop(0.45, '#fc644c');
  gradient.addColorStop(1, '#e9ff6a');
  sourceCtx.fillStyle = gradient;
  sourceCtx.fillRect(0, 0, 960, 620);
  sourceCtx.fillStyle = 'rgba(9,11,16,.72)';
  sourceCtx.beginPath();
  sourceCtx.arc(700, 190, 180, 0, Math.PI * 2);
  sourceCtx.fill();
  sourceCtx.fillStyle = 'rgba(240,244,255,.74)';
  sourceCtx.beginPath();
  sourceCtx.moveTo(120, 480);
  sourceCtx.lineTo(430, 80);
  sourceCtx.lineTo(590, 520);
  sourceCtx.fill();
  runFrame();
}

async function loadFile(file) {
  fileBase = file.name.replace(/\.[^.]+$/, '') || 'polygonalized';
  $('#fileName').textContent = file.name.toUpperCase();
  cancelAnimationFrame(raf);
  if (session) {
    window.polygonalizeClose(session);
    session = null;
  }
  if (activeMediaURL) URL.revokeObjectURL(activeMediaURL);
  activeMediaURL = URL.createObjectURL(file);
  if (file.type.startsWith('video/')) {
    video.src = activeMediaURL;
    await video.play();
    const [width, height] = fit(video.videoWidth, video.videoHeight, 900);
    source.width = width;
    source.height = height;
    session = window.polygonalizeStart(new Uint8Array(width * height * 4), width, height, JSON.stringify(opts()));
    const tick = (now) => {
      if (now - lastFrame > 70) {
        sourceCtx.drawImage(video, 0, 0, width, height);
        runFrame(true);
        lastFrame = now;
      }
      raf = requestAnimationFrame(tick);
    };
    tick(0);
  } else {
    video.removeAttribute('src');
    const image = new Image();
    image.src = activeMediaURL;
    await image.decode();
    const [width, height] = fit(image.naturalWidth, image.naturalHeight);
    source.width = width;
    source.height = height;
    sourceCtx.drawImage(image, 0, 0, width, height);
    runFrame();
  }
}

function rerun() {
  if (video.src) {
    if (session) window.polygonalizeClose(session);
    session = window.polygonalizeStart(new Uint8Array(source.width * source.height * 4), source.width, source.height, JSON.stringify(opts()));
  } else {
    runFrame();
  }
}

function scheduleRerun() {
  clearTimeout(rerunTimer);
  $('#engineStatus').textContent = +$('#triangles').value > 3000 ? 'Preparing high-detail topology…' : 'Go/WASM ready · local mode';
  rerunTimer = setTimeout(() => {
    rerun();
    $('#engineStatus').textContent = 'Go/WASM ready · local mode';
  }, 120);
}

for (const id of ['triangles', 'edgeBias', 'stability']) {
  $(`#${id}`).addEventListener('input', () => {
    $('#trianglesValue').textContent = `${(+$('#triangles').value).toLocaleString()} triangles`;
    $('#edgeValue').textContent = `${$('#edgeBias').value}%`;
    $('#stabilityValue').textContent = `${$('#stability').value}%`;
    scheduleRerun();
  });
}

$('#primitive').addEventListener('change', () => {
  const kind = primitive();
  $('#primitiveUpload').hidden = kind !== 'custom';
  $('#primitiveStatus').textContent = kind === 'custom' ? (customPrimitive ? 'Custom ready' : 'Upload needed') : kind === 'triangle' ? 'Mesh fill' : kind[0].toUpperCase() + kind.slice(1);
  $('#primitiveHint').textContent = kind === 'custom'
    ? 'Transparency is rendered as-is and ignored by the mesh algorithm.'
    : 'Shape selection only affects rendering, never edge sampling.';
  draw();
});

$('#primitiveInput').addEventListener('change', async (event) => {
  const file = event.target.files[0];
  if (!file) return;
  if (customPrimitiveURL) URL.revokeObjectURL(customPrimitiveURL);
  customPrimitiveURL = URL.createObjectURL(file);
  customPrimitiveData = await new Promise((resolve) => {
    const reader = new FileReader();
    reader.onload = () => resolve(reader.result);
    reader.readAsDataURL(file);
  });
  customPrimitive = new Image();
  customPrimitive.src = customPrimitiveURL;
  await customPrimitive.decode();
  $('#primitiveFile').textContent = file.name;
  $('#primitiveStatus').textContent = 'Custom ready';
  draw();
});

$('.segmented').addEventListener('click', (event) => {
  if (!event.target.dataset.view) return;
  view = event.target.dataset.view;
  document.querySelectorAll('.view').forEach((button) => button.classList.toggle('active', button === event.target));
  $('#renderer').textContent = view === 'three' ? 'Three.js mesh preview' : 'Canvas 2D';
  draw();
});

$('#randomize').addEventListener('click', () => {
  seed = Math.floor(Math.random() * 1e9);
  scheduleRerun();
});

$('#download').addEventListener('click', () => {
  if (!mesh) return;
  const kind = primitive();
  let defs = '';
  let shapes;
  if (kind === 'custom' && customPrimitiveData) {
    defs = `<defs><image id="custom-primitive" href="${customPrimitiveData}" width="1" height="1" preserveAspectRatio="xMidYMid meet"/></defs>`;
    shapes = mesh.triangles.map((triangle) => {
      const m = triangleMetrics(triangle);
      const size = Math.sqrt(m.area) * 1.38;
      return `<use href="#custom-primitive" x="${(m.x - size / 2).toFixed(2)}" y="${(m.y - size / 2).toFixed(2)}" width="${size.toFixed(2)}" height="${size.toFixed(2)}"/>`;
    }).join('');
  } else {
    const shapeKind = kind === 'custom' ? 'triangle' : kind;
    shapes = mesh.triangles.map((triangle) => {
      const points = primitivePoints(triangle, shapeKind).map((point) => `${point.x.toFixed(2)},${point.y.toFixed(2)}`).join(' ');
      return `<polygon points="${points}" fill="${triangle.color}"/>`;
    }).join('');
  }
  const blob = new Blob([`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ${mesh.width} ${mesh.height}">${defs}${shapes}</svg>`], { type: 'image/svg+xml' });
  const link = document.createElement('a');
  link.href = URL.createObjectURL(blob);
  link.download = `${fileBase}-polygonalized.svg`;
  link.click();
  setTimeout(() => URL.revokeObjectURL(link.href), 1000);
});

const drop = $('#dropZone');
const input = $('#fileInput');
input.addEventListener('change', () => input.files[0] && loadFile(input.files[0]));
drop.addEventListener('keydown', (event) => {
  if (event.key === 'Enter' || event.key === ' ') {
    event.preventDefault();
    input.click();
  }
});
for (const eventName of ['dragenter', 'dragover']) drop.addEventListener(eventName, (event) => {
  event.preventDefault();
  drop.classList.add('drag');
});
for (const eventName of ['dragleave', 'drop']) drop.addEventListener(eventName, (event) => {
  event.preventDefault();
  drop.classList.remove('drag');
});
drop.addEventListener('drop', (event) => event.dataTransfer.files[0] && loadFile(event.dataTransfer.files[0]));

async function boot() {
  try {
    const go = new Go();
    const result = await WebAssembly.instantiateStreaming(fetch('/assets/polygonalize.wasm'), go.importObject);
    go.run(result.instance);
    while (!window.polygonalizeReady) await new Promise((resolve) => setTimeout(resolve, 10));
    $('#engineDot').classList.add('ready');
    $('#engineStatus').textContent = 'Go/WASM ready · local mode';
    generatedDemo();
  } catch (error) {
    console.error(error);
    $('#engineStatus').textContent = 'WASM unavailable · use server API';
  }
}

boot();
