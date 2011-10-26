package com.dtrules.samples.chipeligibility.app;

import com.dtrules.samples.chipeligibility.app.dataobjects.Job;

public interface EvaluateJob {

	public abstract String getName();
	public abstract String evaluate(int threadnum, ChipApp app, Job job);

}