import { App } from '@tinyhttp/app'
import { createServer } from 'node:http'
import type { AddressInfo } from 'node:net'
import { resolve } from 'node:path'
import sirv from 'sirv'

const app = new App()
const server = createServer(app.handler.bind(app))

app.use(sirv(resolve(import.meta.dirname, '../demo', 'dist')))

server.listen(3000, () => {
  console.log(`Server listening on ${(server.address()! as AddressInfo).port}`)
})
