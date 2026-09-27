struct Uniforms {
    posX: f32,
    posY: f32,
    posZ: f32,
    rightX: f32,
    rightY: f32,
    rightZ: f32,
    upX: f32,
    upY: f32,
    upZ: f32,
    fwdX: f32,
    fwdY: f32,
    fwdZ: f32,
    scale: f32,
    octaves: f32,
    alpha: f32,
    beta: f32,
    density: f32,
    steps: f32,
    iso: f32,
    resX: f32,
    resY: f32,
    time: f32,
    gain: f32,
    drift: f32,
}

@group(0) @binding(0) var<uniform> u: Uniforms;
@group(0) @binding(1) var<storage, read> table: array<vec4<f32>>;

const N: f32 = 4096.0;
const BM: i32 = 255;
const TAN_HALF_FOV: f32 = 0.6;
const EPS: f32 = 0.002;

fn sCurve(t: f32) -> f32 {
    return t * t * (3.0 - 2.0 * t);
}

fn lerp(t: f32, a: f32, b: f32) -> f32 {
    return a + t * (b - a);
}

fn at3(rx: f32, ry: f32, rz: f32, q: vec4<f32>) -> f32 {
    return rx * q.y + ry * q.z + rz * q.w;
}

fn noise3(v: vec3<f32>) -> f32 {
    let tx = v.x + N;
    let bx0 = i32(tx) & BM;
    let bx1 = (bx0 + 1) & BM;
    let rx0 = tx - floor(tx);
    let rx1 = rx0 - 1.0;

    let ty = v.y + N;
    let by0 = i32(ty) & BM;
    let by1 = (by0 + 1) & BM;
    let ry0 = ty - floor(ty);
    let ry1 = ry0 - 1.0;

    let tz = v.z + N;
    let bz0 = i32(tz) & BM;
    let bz1 = (bz0 + 1) & BM;
    let rz0 = tz - floor(tz);
    let rz1 = rz0 - 1.0;

    let i = i32(table[bx0].x);
    let j = i32(table[bx1].x);

    let b00 = i32(table[i + by0].x);
    let b10 = i32(table[j + by0].x);
    let b01 = i32(table[i + by1].x);
    let b11 = i32(table[j + by1].x);

    let sx = sCurve(rx0);
    let sy = sCurve(ry0);
    let sz = sCurve(rz0);

    var a = lerp(sx, at3(rx0, ry0, rz0, table[b00 + bz0]), at3(rx1, ry0, rz0, table[b10 + bz0]));
    var b = lerp(sx, at3(rx0, ry1, rz0, table[b01 + bz0]), at3(rx1, ry1, rz0, table[b11 + bz0]));
    let c = lerp(sy, a, b);

    a = lerp(sx, at3(rx0, ry0, rz1, table[b00 + bz1]), at3(rx1, ry0, rz1, table[b10 + bz1]));
    b = lerp(sx, at3(rx0, ry1, rz1, table[b01 + bz1]), at3(rx1, ry1, rz1, table[b11 + bz1]));
    let d = lerp(sy, a, b);

    return lerp(sz, c, d);
}

fn fbm3(p: vec3<f32>) -> f32 {
    var sum = 0.0;
    var amplitude = 1.0;
    var pos = p;

    for (var i = 0.0; i < u.octaves; i = i + 1.0) {
        sum = sum + noise3(pos) / amplitude;
        amplitude = amplitude * u.alpha;
        pos = pos * u.beta;
    }

    return sum;
}

fn field(p: vec3<f32>) -> f32 {
    return fbm3(p * u.scale + vec3<f32>(0.0, 0.0, u.time * u.drift));
}

fn gradient(p: vec3<f32>) -> vec3<f32> {
    let e = EPS / max(u.scale, 0.001);

    let dx = field(p + vec3<f32>(e, 0.0, 0.0)) - field(p - vec3<f32>(e, 0.0, 0.0));
    let dy = field(p + vec3<f32>(0.0, e, 0.0)) - field(p - vec3<f32>(0.0, e, 0.0));
    let dz = field(p + vec3<f32>(0.0, 0.0, e)) - field(p - vec3<f32>(0.0, 0.0, e));

    return vec3<f32>(dx, dy, dz);
}

fn boxRange(ro: vec3<f32>, rd: vec3<f32>, halfExtent: f32) -> vec2<f32> {
    let safe = select(rd, vec3<f32>(1e-6), abs(rd) < vec3<f32>(1e-6));
    let inv = 1.0 / safe;
    let extent = vec3<f32>(halfExtent);

    let t0 = (-extent - ro) * inv;
    let t1 = (extent - ro) * inv;
    let lo = min(t0, t1);
    let hi = max(t0, t1);

    return vec2<f32>(max(max(lo.x, lo.y), lo.z), min(min(hi.x, hi.y), hi.z));
}

@vertex
fn vsMain(@builtin(vertex_index) vertexIndex: u32) -> @builtin(position) vec4<f32> {
    var positions = array<vec2<f32>, 3>(
        vec2<f32>(-1.0, -1.0),
        vec2<f32>(3.0, -1.0),
        vec2<f32>(-1.0, 3.0),
    );

    return vec4<f32>(positions[vertexIndex], 0.0, 1.0);
}

@fragment
fn raymarch(@builtin(position) fragPos: vec4<f32>) -> @location(0) vec4<f32> {
    let background = vec3<f32>(0.03, 0.03, 0.04);

    let ndcX = fragPos.x / u.resX * 2.0 - 1.0;
    let ndcY = 1.0 - fragPos.y / u.resY * 2.0;

    let ro = vec3<f32>(u.posX, u.posY, u.posZ);
    let fwd = vec3<f32>(u.fwdX, u.fwdY, u.fwdZ);
    let right = vec3<f32>(u.rightX, u.rightY, u.rightZ);
    let up = vec3<f32>(u.upX, u.upY, u.upZ);

    let aspect = u.resX / max(u.resY, 1.0);
    let rd = normalize(fwd + right * (ndcX * aspect * TAN_HALF_FOV) + up * (ndcY * TAN_HALF_FOV));

    let range = boxRange(ro, rd, 1.0);
    var t = max(range.x, 0.0);
    let tEnd = range.y;

    if (tEnd <= t || u.octaves < 1.0) {
        return vec4<f32>(background, 1.0);
    }

    let steps = i32(clamp(u.steps, 8.0, 512.0));
    let dt = (tEnd - t) / f32(steps);

    let fogColor = vec3<f32>(0.30, 0.33, 0.40);
    var accum = vec3<f32>(0.0);
    var trans = 1.0;

    for (var i = 0; i < steps; i = i + 1) {
        let pos = ro + rd * t;
        let value = field(pos);
        let above = value - u.iso;

        if (above > 0.0) {
            let nrm = normalize(-gradient(pos));
            let lightDir = normalize(vec3<f32>(0.4, 0.7, 0.5));

            let diffuse = clamp(dot(nrm, lightDir), 0.0, 1.0);
            let specular = pow(clamp(dot(reflect(-lightDir, nrm), -rd), 0.0, 1.0), 24.0);

            let shade = clamp(value * u.gain + 0.5, 0.0, 1.0);
            let low = vec3<f32>(0.15, 0.20, 0.45);
            let high = vec3<f32>(0.95, 0.80, 0.55);
            let albedo = mix(low, high, shade);

            let surface = albedo * (0.2 + 0.8 * diffuse) + vec3<f32>(specular * 0.4);
            accum = accum + surface * trans;
            trans = 0.0;
            break;
        }

        let a = clamp(above * 4.0 + 0.35, 0.0, 1.0) * dt * u.density;
        accum = accum + fogColor * a * trans;
        trans = trans * (1.0 - a);
        t = t + dt;

        if (trans < 0.01) {
            break;
        }
    }

    return vec4<f32>(clamp(accum + trans * background, vec3<f32>(0.0), vec3<f32>(1.0)), 1.0);
}
