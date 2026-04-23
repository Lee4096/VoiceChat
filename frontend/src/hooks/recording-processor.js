class RecordingProcessor extends AudioWorkletProcessor {
  constructor() {
    super()
    this.recording = true
    this.port.onmessage = (event) => {
      if (event.data === 'stop') {
        this.recording = false
      }
    }
  }

  process(inputs, outputs, parameters) {
    if (!this.recording) {
      return true
    }

    const input = inputs[0]
    if (input && input.length > 0) {
      const channelData = input[0]
      if (channelData && channelData.length > 0) {
        const buffer = new Float32Array(channelData)
        this.port.postMessage({ buffer })
      }
    }
    return true
  }
}

registerProcessor('recording-processor', RecordingProcessor)
