import { Socket } from 'socket.io-client'
import { getSocket } from './socket'

let wss: Socket | undefined
const setup = () => {
  console.log('setting up socket-io client...')
  
  if (!wss) {
    wss = getSocket()
  }
  wss.on('connection', (socket) => {
    console.log('socket connection established! ID:', socket.id)
  })
  wss.on('data', (e) => {
    console.log('received data:', e)
    postMessage(e)    
  })
  wss.on('test', () => {
    postMessage('hello from worker')
  })
}
setup()
onmessage = (event: MessageEvent<{ ping?: boolean, data?: any }>) => {
  console.log('received message: ', event.data)
  if (event.data.ping) {
    wss?.emit('test', 'ping')
  }
}