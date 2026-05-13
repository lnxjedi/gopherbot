const { PubSub } = require('@google-cloud/pubsub');
const pubsub = new PubSub();

exports.webhookIngest = async (req, res) => {
  try {
    const topicName = process.env.JOB_WEBHOOK_TOPIC;
    
    // req.rawBody contains the unparsed bytes of the request
    if (!req.rawBody || req.rawBody.length === 0) {
      return res.status(400).send('Empty payload');
    }

    // Publish the raw buffer directly
    await pubsub.topic(topicName).publishMessage({ data: req.rawBody });

    res.status(202).send('Accepted');
  } catch (error) {
    console.error(`Failed to publish to ${process.env.JOB_WEBHOOK_TOPIC}:`, error);
    res.status(500).send('Internal Server Error');
  }
};