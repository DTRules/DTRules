package com.dtrules.samples.bookpreview.app;

import com.dtrules.samples.bookpreview.datamodel.Request;


public interface EvaluateJob {

	public abstract String getName();
	public abstract String evaluate(int threadnum, BookPreviewApp app, Request request);

}