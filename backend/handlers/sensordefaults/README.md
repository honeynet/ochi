Offline fallback for the default Glutton config and rules.

The sensor config and rules endpoints fetch these files from the Glutton repository at the
pinned revision in `sensorconfig.go` (`gluttonRef`), so they are not a second copy that has
to be kept in step by hand. These embedded files are served only when that fetch cannot be
made, so the endpoints keep working with no network.

Copied from mushorg/glutton at commit 54fd4c9f578bbb9f1d77093b4ddeb04b28771422
(https://github.com/mushorg/glutton/tree/main/config). When the pinned revision is moved,
refresh these files from the same revision so the fallback matches what is fetched.
