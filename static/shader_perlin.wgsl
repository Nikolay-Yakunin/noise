struct Params {
    width:   u32
    height:  u32
    scale:   u32
    octaves: u32
    alpha:   f32
    beta:    f32
    seed:    u32
    _pad:    u32
}

@group(0) @binding(0) var<uniform> params: Params;
@group(0) @binding(1) var<storage, read_write> output: array<f32>;
@group(0) @binding(2) var<storage, read> perm: array<i32>;
@group(0) @binding(3) var<storage, read> grads: array<vec2<f32>>;

const N: f32 = 4096.0;
const BM: i32 = 255;

fn sCurve(t: f32) -> f32 {
    return t * t * (3.0 - 2.0 * t);
}

fn lerp(t: f32, a: f32, b: f32) -> f32 {
    return a + t * (b - a);
}

fn at2(rx: f32, ry: f32, q: vec2<f32>) -> f32 {
    return rx * q.x + ry * q.y;
}

fn noise2(v: vec2<f32>) -> f32 {
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

    let i = perm[bx0];
    let j = perm[bx1];

    let b00 = perm[i + by0];
    let b10 = perm[j + by0];
    let b01 = perm[i + by1];
    let b11 = perm[j + by1];

    let sx = sCurve(rx0);
    let sy = sCurve(ry0);

    let a = lerp(sx, at2(rx0, ry0, grads[b00]), at2(rx1, ry0, grads[b10]));
    let b = lerp(sx, at2(rx0, ry1, grads[b01]), at2(rx1, ry1, grads[b11]));

    return lerp(sy, a, b);
}

@compute @workgroup_size(16, 16)
fn main(@builtin(global_invocation_id) id: vec3<u32>) {
    let x = id.x;
    let y = id.y;

    if (x >= params.width || y >= params.height) {
        return;
    }

    let scale = f32(params.scale);
    var px = vec2<f32>(f32(x) / scale, f32(y) / scale);

    var total: f32 = 0.0;
    var amplitude: f32 = 1.0;

    for (var i: u32 = 0u; i < params.octaves; i = i + 1u) {
        total = total + noise2(px) / amplitude;
        amplitude = amplitude * params.alpha;
        px = px * params.beta;
    }

    let idx = y * params.width + x;
    output[idx] = total;
}
