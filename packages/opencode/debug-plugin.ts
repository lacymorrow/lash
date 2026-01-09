
import * as LashCore from "@lash-cli/core"

console.log("Exports of @lash-cli/core:")
console.log(Object.keys(LashCore))

if (typeof LashCore.lashCore === "function") {
    console.log("lashCore is a function")
} else {
    console.log("lashCore is NOT a function", typeof LashCore.lashCore)
}
