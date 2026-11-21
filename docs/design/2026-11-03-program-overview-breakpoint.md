# Program Overview responsive composition

The approved Program HTML stacks the coverage/recent-assessment column before
the attention rail at `max-width: 1170px`, and displays the rail's attention
and setup cards side by side from 761px through that breakpoint. The runtime
kept the columns side by side until 1023px and used a single rail column at
768px. Align both replacement layouts to the source while retaining all real
data, actions, accessibility, uncertainty and authorization contracts.

Compare the same Program source and runtime at the five approved widths and
both runtime themes, plus the transition boundaries. Verify section geometry,
first useful content, keyboard and touch actions, long content, and unavailable
and partial states. Source example values and unsupported control-area results
must not be copied. This is partial V2 integration, not screen acceptance.

At widths just above the breakpoint, the existing current-position action
overflowed its narrow rail because its header kept an unwrapped horizontal
layout. Stack that header within the rail and constrain the existing action to
the available width; preserve its authority and focus behavior.
