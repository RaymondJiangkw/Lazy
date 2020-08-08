# Utils

Utils is a *GO* package, which integrates many little convenient functions *(nearly all concurrency safe)* for usage.

I know it would be better to separate tons of functions into different packages according to their area of usage. But, I could not come up with a good name for each of them. *utils* is short and easy to remember. However, something like *utils_web*, *utils_io* is not so elegant. And names like *ioutils* have been used.

Thus, in order to speed up, I abandon general `init` function, and switch to *sync.Once*.

Perhaps, in the future, I may decide to separate this package into smaller ones as it becomes redundant.