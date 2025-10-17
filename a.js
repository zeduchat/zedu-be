const { readFileSync } = require("fs")
const content = String(readFileSync("output.json"))

const json = JSON.parse(content)

console.log(typeof(json), Object.keys(json.data))

console.log(json.data.map((item) => ({name: item.name, type: typeof(item)})))


