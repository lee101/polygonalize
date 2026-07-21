const $ = (q) => document.querySelector(q);
const canvas = $('#output');
const ctx = canvas.getContext('2d', { alpha: false });
const source = document.createElement('canvas');
const sourceCtx = source.getContext('2d', { willReadFrequently: true });
const video = $('#video');
let mesh = null, seed = 17, session = null, raf = 0, lastFrame = 0, fileBase = 'polygonalized', view = 'flat';
let three = null;

const opts = () => ({points:+$('#points').value, edgeBias:+$('#edgeBias').value/100, stability:+$('#stability').value/100, seed});
const fit = (w,h,max=1100) => { const scale=Math.min(1,max/Math.max(w,h)); return [Math.max(2,Math.round(w*scale)),Math.max(2,Math.round(h*scale))]; };

function imageData() { return sourceCtx.getImageData(0,0,source.width,source.height); }
function runFrame(stable=false) {
  if (!window.polygonalizeReady || !source.width) return;
  const data=imageData(), json=stable
    ? window.polygonalizeFrame(session,data.data,data.width,data.height,JSON.stringify(opts()))
    : window.polygonalizeImage(data.data,data.width,data.height,JSON.stringify(opts()));
  mesh=JSON.parse(json); draw(); $('#stats').textContent=`${mesh.triangles.length} triangles`;
}
function draw() {
  if (!mesh) return;
  if (view==='three') { showThree(); return; }
  $('#threeStage').hidden=true;canvas.hidden=false;canvas.width=mesh.width;canvas.height=mesh.height;
  ctx.fillStyle='#111820';ctx.fillRect(0,0,canvas.width,canvas.height);
  for(const tri of mesh.triangles){const a=mesh.points[tri.a],b=mesh.points[tri.b],c=mesh.points[tri.c];ctx.beginPath();ctx.moveTo(a.x,a.y);ctx.lineTo(b.x,b.y);ctx.lineTo(c.x,c.y);ctx.closePath();ctx.fillStyle=tri.color;ctx.fill();ctx.strokeStyle=view==='wire'?'rgba(240,244,255,.34)':tri.color;ctx.lineWidth=view==='wire'?.7:1;ctx.stroke();}
}
async function showThree(){
  canvas.hidden=true;const host=$('#threeStage');host.hidden=false;$('#renderer').textContent='Three.js WebGL';
  if(!three){
    const THREE=await import('https://cdn.jsdelivr.net/npm/three@0.179.1/build/three.module.js');
    const renderer=new THREE.WebGLRenderer({antialias:true});renderer.setPixelRatio(Math.min(devicePixelRatio,2));renderer.setClearColor(0x10151c);host.replaceChildren(renderer.domElement);
    const scene=new THREE.Scene(),camera=new THREE.PerspectiveCamera(40,1,.1,100);camera.position.set(0,0,3.1);scene.add(new THREE.AmbientLight(0xffffff,1.5));
    three={THREE,renderer,scene,camera,obj:null};
    host.addEventListener('pointermove',e=>{const r=host.getBoundingClientRect();camera.position.x=((e.clientX-r.left)/r.width-.5)*.7;camera.position.y=-((e.clientY-r.top)/r.height-.5)*.45;camera.lookAt(0,0,0);});
    new ResizeObserver(()=>{const r=host.getBoundingClientRect();renderer.setSize(r.width,r.height,false);camera.aspect=r.width/r.height;camera.updateProjectionMatrix()}).observe(host);
    const loop=()=>{if(three){three.renderer.render(three.scene,three.camera);requestAnimationFrame(loop)}};loop();
  }
  const {THREE}=three;if(three.obj){three.scene.remove(three.obj);three.obj.geometry.dispose();three.obj.material.dispose()}
  const pos=[],colors=[],color=new THREE.Color();for(const t of mesh.triangles){for(const i of [t.a,t.b,t.c]){const p=mesh.points[i];const x=p.x/mesh.width-.5,y=.5-p.y/mesh.height;const z=Math.sin(x*5)*.035+Math.cos(y*6)*.035;pos.push(x*2,y*2,z);color.set(t.color);colors.push(color.r,color.g,color.b)}}
  const g=new THREE.BufferGeometry();g.setAttribute('position',new THREE.Float32BufferAttribute(pos,3));g.setAttribute('color',new THREE.Float32BufferAttribute(colors,3));g.computeVertexNormals();three.obj=new THREE.Mesh(g,new THREE.MeshStandardMaterial({vertexColors:true,roughness:.88,metalness:.04,side:THREE.DoubleSide}));three.scene.add(three.obj);
}
function generatedDemo(){source.width=960;source.height=620;const g=sourceCtx.createLinearGradient(0,0,960,620);g.addColorStop(0,'#7b52e6');g.addColorStop(.45,'#fc644c');g.addColorStop(1,'#e9ff6a');sourceCtx.fillStyle=g;sourceCtx.fillRect(0,0,960,620);sourceCtx.fillStyle='rgba(9,11,16,.72)';sourceCtx.beginPath();sourceCtx.arc(700,190,180,0,Math.PI*2);sourceCtx.fill();sourceCtx.fillStyle='rgba(240,244,255,.74)';sourceCtx.beginPath();sourceCtx.moveTo(120,480);sourceCtx.lineTo(430,80);sourceCtx.lineTo(590,520);sourceCtx.fill();runFrame();}
async function loadFile(file){fileBase=file.name.replace(/\.[^.]+$/,'')||'polygonalized';$('#fileName').textContent=file.name.toUpperCase();cancelAnimationFrame(raf);if(session){window.polygonalizeClose(session);session=null}const url=URL.createObjectURL(file);if(file.type.startsWith('video/')){video.src=url;await video.play();const [w,h]=fit(video.videoWidth,video.videoHeight,900);source.width=w;source.height=h;session=window.polygonalizeStart(new Uint8Array(w*h*4),w,h,JSON.stringify(opts()));const tick=(now)=>{if(now-lastFrame>70){sourceCtx.drawImage(video,0,0,w,h);runFrame(true);lastFrame=now}raf=requestAnimationFrame(tick)};tick(0)}else{const img=new Image();img.src=url;await img.decode();const [w,h]=fit(img.naturalWidth,img.naturalHeight);source.width=w;source.height=h;sourceCtx.drawImage(img,0,0,w,h);runFrame()}URL.revokeObjectURL(url)}
function rerun(){if(video.src){if(session)window.polygonalizeClose(session);session=window.polygonalizeStart(new Uint8Array(source.width*source.height*4),source.width,source.height,JSON.stringify(opts()))}else runFrame()}
for(const id of ['points','edgeBias','stability']){$(`#${id}`).addEventListener('input',()=>{$('#pointsValue').textContent=`${$('#points').value} points`;$('#edgeValue').textContent=`${$('#edgeBias').value}%`;$('#stabilityValue').textContent=`${$('#stability').value}%`;rerun()})}
$('.segmented').addEventListener('click',e=>{if(!e.target.dataset.view)return;view=e.target.dataset.view;document.querySelectorAll('.view').forEach(b=>b.classList.toggle('active',b===e.target));$('#renderer').textContent=view==='three'?'Three.js WebGL':'Canvas 2D';draw()});
$('#randomize').addEventListener('click',()=>{seed=Math.floor(Math.random()*1e9);rerun()});
$('#download').addEventListener('click',()=>{if(!mesh)return;const polygons=mesh.triangles.map(t=>{const p=[mesh.points[t.a],mesh.points[t.b],mesh.points[t.c]].map(v=>`${v.x.toFixed(2)},${v.y.toFixed(2)}`).join(' ');return `<polygon points="${p}" fill="${t.color}"/>`}).join('');const blob=new Blob([`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ${mesh.width} ${mesh.height}">${polygons}</svg>`],{type:'image/svg+xml'});const a=document.createElement('a');a.href=URL.createObjectURL(blob);a.download=`${fileBase}-polygonalized.svg`;a.click();setTimeout(()=>URL.revokeObjectURL(a.href),1000)});
const drop=$('#dropZone'),input=$('#fileInput');input.addEventListener('change',()=>input.files[0]&&loadFile(input.files[0]));drop.addEventListener('keydown',e=>{if(e.key==='Enter'||e.key===' '){e.preventDefault();input.click()}});for(const ev of ['dragenter','dragover'])drop.addEventListener(ev,e=>{e.preventDefault();drop.classList.add('drag')});for(const ev of ['dragleave','drop'])drop.addEventListener(ev,e=>{e.preventDefault();drop.classList.remove('drag')});drop.addEventListener('drop',e=>e.dataTransfer.files[0]&&loadFile(e.dataTransfer.files[0]));
async function boot(){try{const go=new Go();const result=await WebAssembly.instantiateStreaming(fetch('/assets/polygonalize.wasm'),go.importObject);go.run(result.instance);while(!window.polygonalizeReady)await new Promise(r=>setTimeout(r,10));$('#engineDot').classList.add('ready');$('#engineStatus').textContent='Go/WASM ready · local mode';generatedDemo()}catch(err){console.error(err);$('#engineStatus').textContent='WASM unavailable · use server API'}}boot();
