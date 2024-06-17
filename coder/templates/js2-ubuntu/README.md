# Ubuntu Linux on JetStream 2

This template launches a VM running Ubuntu on JetStream2.

## Known Issues

### "workspace build failed" when stopping the workspace

When you stop the workspace, you'll get an error message that the "workspace
build failed". You can ignore it -- to fully shutdown the worspace, you need
click "retry" in the top-right corner. The second time, the "build" should 
succeed.