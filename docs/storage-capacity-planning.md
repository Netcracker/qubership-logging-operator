## Table of Contents
- [Storage capacity planning](#storage-capacity-planning)
  * [Log `storm`](#log--storm-)
  * [Indices with rotation by size](#indices-with-rotation-by-size)
  * [Indices with rotation by time or messages count](#indices-with-rotation-by-time-or-messages-count)

## Storage capacity planning

To calculate the total storage size required to store logs from you environment need to calculate
how many logs planning store in **each** Stream/Index.

Default Indices:

* Default index set (`All messages` Stream)
* Audit index set (`Audit logs` Stream)
* Graylog Events (`All events` Stream)
* Graylog System Events (`All system events` Stream)
* Kubernetes events index set (`Kubernetes events` Stream)

Except default Streams/Indices products/projects can create it's own Streams/Indices. They also should
be include in the calculation.

All received logs Graylog saved in OpenSearch/ElasticSearch. The OpenSearch/ElasticSearch has protection
to prevent OpenSearch nodes from running out of disk space.

By default, OpenSearch marks all indices as read-only if data usage reaches a set threshold.
This threshold by default set as **95%** of available disk or volume space.

Next, for the expected log size for **N** days, you have to add **15-25%** of free space to avoid problems
with the read-only index and log rotation.

So the resulting formula will be something like this:

```bash
(Expected size of Index 1 + Expected size of Index 2 + ... + Expected size of Index N) / 0.80 = Total size of storage
```

Based on the calculated size, you have to set other parameters, like max index size, count of indices,
and so on.
In the case of storing a big log count (by size), better to increase the max index size
from 1 Gb to 5-10-20 Gb.

### Log `storm`

Do not forget to add in your calculations reserve for case of `log storm`.
In some cases, problems with one service can lead to increased log generation in other related services.

THe simple example:

* there are 20 Java services that are using PostgreSQL
* all these services generating about 0-100 log lines per minute
* but in moment PostgreSQL become unavailable (due the internal or network problems)
* all 20 Java services may start generate in their logs giant stacktraces with errors about PostgreSQL unavailability
* log generation for these 20 Java services can increase in **10-100 times** for PostgreSQL unavailability period

In this example, in the case of problems with PostgreSQL in Graylog may be send 200000 logs per minute
instead of 2000 per minute which we expect during the normal work.

### Indices with rotation by size

You **have to estimate** how many logs per day your Graylog received for each Stream/Index.

**Note:** For `All messages` Stream the simplest way is using information from Graylog UI,
navigate to `Graylog UI -> System -> Overview` and check values in histogram.

For example, if on this page you see:

* 50 GB for 1 day ago
* 100 GB for 2 days ago

you can calculate the average value or select the most pessimistic value.
Next, need to multiply the selected value by the count of days.

**Note:** You may add reserve for log `storm`, but it's optional unlike for case using rotation
by time or message count.

So the resulting formula will be something like this:

```bash
X Gb (Logs size per day) * Y days = Total size of Index
```

For example:

```bash
100 Gb * 7 days = 700 Gb of Index
```

Also, you have to keep in mind that in Graylog you have some Streams and Indices.
So this calculation must be done for all existing Indices.

As an alternative you can use the metrics like:

```prometheus
avg(gl_input_read_bytes_one_sec[1h])
```

or (it's not a PromQL expression, just an example):

```prometheus
rate(gl_input_read_bytes_total) / rate(gl_input_incoming_messages_total)
```

But in this case, you have to be very accurate because the load can change over time.
You can't calculate it only for one time point, you have to calculate for the time range
and next calculate the average or max value.

### Indices with rotation by time or messages count

In the case, if you want to use Streams with Indices rotated by time or message count,
you **have to be extremely careful** and estimate the expected incoming log flow very well.

Unlike rotation by size, using the rotation by time or message count can lead to index/disk overflow
if expected logs size will be calculated incorrect. It may affect logs storing in OpenSearch
and in all other Streams.

There is no a simple and unambiguous formula to calculate the required storage size for Indices
with rotation by time or messages count. But you can try to start from the formula:

* Rotation by time

  ```bash
  X Gb (Logs size per day) * Y days + Z Gb (log storm reserve) = Total size of Index
  ```

* Rotation by message count:

  ```bash
  X Kb (average 1 message size) * Y messages count + Z Gb (log storm reserve) = Total size of Index
  ```

  **Note:** Please keep in mind, that each message has a lot of meta information (additional fields),
  like `namespace`, `pod`, `container`, labels and others that also should be include in the 1 message size.
