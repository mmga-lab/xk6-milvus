// Test vector generation
function generateDissimilarVector(dim, seed) {
    const vector = [];
    for (let i = 0; i < dim; i++) {
        // Completely different pattern
        vector.push(Math.cos(seed * 2 + i * 0.3) * 0.5 + 0.5);
    }
    return vector;
}

const v = generateDissimilarVector(128, 0);
console.log('First 5 elements:', v.slice(0, 5));
console.log('Element types:', v.slice(0, 5).map(x => typeof x));
console.log('Element 0:', v[0], 'Type:', typeof v[0]);

const numericVector = v.map(v => Number(v));
console.log('After Number() - First 5:', numericVector.slice(0, 5));
console.log('After Number() - Types:', numericVector.slice(0, 5).map(x => typeof x));
