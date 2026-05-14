const { PubSub } = require('@google-cloud/pubsub');
const pubsub = new PubSub();

const MAX_BODY_SIZE = 4096;
const UUID_PREFIX_RE = /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$/;
const UUID_PREFIX_LEN = 36;

exports.webhookIngest = async (req, res) => {
  try {
    const topicName = process.env.JOB_WEBHOOK_TOPIC;

    if (!topicName) {
      console.error('JOB_WEBHOOK_TOPIC is not set');
      return res.status(500).send('Server misconfiguration');
    }
    
    // req.rawBody contains the unparsed bytes of the request
    if (!req.rawBody || req.rawBody.length === 0) {
      return res.status(400).send('Empty payload');
    }

    if (req.rawBody.length > MAX_BODY_SIZE) {
      return res.status(413).send('Payload too large');
    }

    if (req.rawBody.length < UUID_PREFIX_LEN) {
      return res.status(400).send('Invalid payload format');
    }

    const uuidPrefix = req.rawBody.subarray(0, UUID_PREFIX_LEN).toString('ascii');
    if (!UUID_PREFIX_RE.test(uuidPrefix)) {
      return res.status(400).send('Invalid payload format');
    }

    if (req.rawBody.length > UUID_PREFIX_LEN && req.rawBody[UUID_PREFIX_LEN] !== 0x20) {
      return res.status(400).send('Invalid payload format');
    }

    // Publish the raw buffer directly
    await pubsub.topic(topicName).publishMessage({ data: req.rawBody });

    res.status(202).send('Accepted');
  } catch (error) {
    console.error(`Failed to publish to ${process.env.JOB_WEBHOOK_TOPIC}:`, error);
    res.status(500).send('Internal Server Error');
  }
};