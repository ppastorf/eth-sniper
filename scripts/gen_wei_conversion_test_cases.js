// https://converter.murkin.me/
// console:

const rand = (min, max, decimals) => {
    const str = (Math.random() * (max - min) + min).toFixed(decimals);
  
    return parseFloat(str);
}

const genrand = (n, range, decimals) => {
    vals = []
    for (i=0; i < n; i++) vals.push(rand(range[0], range[1], decimals))
    return vals
}

const sleep = ms => new Promise(r => setTimeout(r, ms));

const gentests = () => ethers.map(ether => {
    document.getElementById("value-input").value = ether
    updateValues(); sleep(10)
    wei = document.getElementById("wei").value
    console.log(`{${wei}, ${ether}},`)
});

const ethers = [
    [2, [0.00, 0.0010], 4],
    [2, [0.00, 0.0010], 18],
    [2, [0.00, 0.10], 4],
    [2, [0.00, 0.10], 18],
    [2, [0.00, 1.00], 18],
    [2, [0.00, 1.00], 4],
    [2, [0.00, 1.00], 18],
    [2, [0.00, 100.00], 4],
    [2, [0.00, 100.00], 18],
    [2, [0.00, 10000.00], 4],
    [2, [0.00, 10000.00], 18],
    [2, [0.00, 1000000.00], 4],
    [2, [0.00, 1000000.00], 18],
    [2, [0.00, 210000000.00], 4],
    [2, [0.00, 210000000.00], 18]
].map(c => genrand(...c)).flat()

gentests()