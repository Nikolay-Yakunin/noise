struct Params {
    width: u32,
    height: u32,
    scale: f32,
    persistence: f32,
    octaves: u32,
}

@group(0) @binding(0) var<uniform> params: Params;
@group(0) @binding(1) var<storage, read_write> output: array<f32>;

fn intNoise(x: i32) -> f32 {
    var n = (x << 13) ^ x;
    let x3 = n * n * n;
    var hash = (x3 * 15731 + 789221) * n + 1376312589;
    hash = hash & 0x7fffffff;
    return 1.0 - f32(hash) / 1073741824.0;
}

fn noise1(x: i32, y: i32) -> f32 {
    return intNoise(x + y * 57);
}

fn smoothNoise(x: i32, y: i32) -> f32 {
    let corners = (
        noise1(x-1, y-1) + noise1(x+1, y-1) + 
        noise1(x-1, y+1) + noise1(x+1, y+1)
    ) / 16.0;
    let sides = (
        noise1(x-1, y) + noise1(x+1, y) + 
        noise1(x, y-1) + noise1(x, y+1)
    ) / 8.0;
    let center = noise1(x, y) / 4.0;
    return corners + sides + center;
}

fn interpolate(a: f32, b: f32, x: f32) -> f32 {
    let ft = x * 3.14159265;
    let f = (1.0 - cos(ft)) * 0.5;
    return a * (1.0 - f) + b * f;
}

fn interpolatedNoise(x: f32, y: f32) -> f32 {
    let integerX = i32(x);
    let fractionalX = x - f32(integerX);
    let integerY = i32(y);
    let fractionalY = y - f32(integerY);
    
    let v1 = smoothNoise(integerX, integerY);
    let v2 = smoothNoise(integerX + 1, integerY);
    let v3 = smoothNoise(integerX, integerY + 1);
    let v4 = smoothNoise(integerX + 1, integerY + 1);
    
    let i1 = interpolate(v1, v2, fractionalX);
    let i2 = interpolate(v3, v4, fractionalX);
    return interpolate(i1, i2, fractionalY);
}

@compute @workgroup_size(16, 16)
fn main(@builtin(global_invocation_id) id: vec3<u32>) {
    let x = id.x;
    let y = id.y;
    
    if (x >= params.width || y >= params.height) {
        return;
    }
    
    let px = f32(x) / params.scale;
    let py = f32(y) / params.scale;
    
    var total: f32 = 0.0;
    var frequency: f32 = 1.0;
    var amplitude: f32 = 1.0;
    
    for (var i: u32 = 0; i < params.octaves; i = i + 1) {
        total = total + interpolatedNoise(px * frequency, py * frequency) * amplitude;
        frequency = frequency * 2.0;
        amplitude = amplitude * params.persistence;
    }
    
    let idx = y * params.width + x;
    output[idx] = total;
}
