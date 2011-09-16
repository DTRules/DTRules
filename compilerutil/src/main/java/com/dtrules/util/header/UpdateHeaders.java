package com.dtrules.util.header;

import java.io.BufferedReader;
import java.io.File;
import java.io.FileInputStream;
import java.io.FileOutputStream;
import java.io.FileReader;
import java.io.FileWriter;
import java.io.InputStream;
import java.io.OutputStream;
import java.io.Reader;
import java.io.Writer;
import java.text.SimpleDateFormat;
import java.util.Date;

public class UpdateHeaders {

	public void update(String directory) throws Exception{
		File f = new File(directory);
		updatefiles(1, f);
	}
	
	private void updatefiles(int level, File f) throws Exception {
		if(f == null){
			return;
		}else if(f.isDirectory()){
			System.out.printf("%"+level+"s [%s]\n"," ",f.getName());
			File files[] = f.listFiles();
			for(File file : files){
				updatefiles(level+1, file);
			}
		}else if(f.getName().endsWith("java") ){
			updateHeader(level, f);
			move(f);
		}
	}
	
	private void move(File f) throws Exception {
		File t = new File(f.getAbsolutePath()+".tmp");
		InputStream  ts = new FileInputStream(t);
		OutputStream fs = new FileOutputStream(f);
		byte[] buf = new byte[10240];
		int len;
		while((len = ts.read(buf))>0){
			fs.write(buf,0,len);
		}
		ts.close();
		fs.close();
		t.delete();
	}
	
	
	private void updateHeader(int level, File f){
		Reader r	= null;
		Writer out 	= null;
		try{
			r   = new FileReader(f);
			out = new FileWriter(f.getAbsolutePath()+".tmp");
		}catch(Exception e){
			System.out.printf("%"+level+"s Could not find '%s'\n"," ",f.getName());
			return;
		}
		BufferedReader br = new BufferedReader(r);
		String line;
		try {
			line = br.readLine();
			
			if(line != null && line.trim().startsWith("/**")){
				System.out.printf("%"+level+"s %s -- has header\n"," ",f.getName());
				while(line != null && !line.contains("*/")){
					line = br.readLine();
				}
				if(line !=null){
					int after = line.indexOf("*/")+2;
					if(after< line.length()){
						line = line.substring(line.indexOf("*/")+2);
					}else{
						line = br.readLine();
					}
				}
			}else{				
				System.out.printf("%"+level+"s %s\n"," ",f.getName());
			}
		
			newHeader(f, out);
			
			while(line!=null){
				out.write(line);
				out.write("\r\n");
				line = br.readLine();
			}
			br.close();
			out.close();
		} catch (Exception e) {
			System.out.println(e.toString());
			e.printStackTrace();
		}
		
	}
	
	private void newHeader(File f, Writer out) throws Exception {
		SimpleDateFormat sdf = new SimpleDateFormat("EEE MMM d yyyy ");
		SimpleDateFormat fY  = new SimpleDateFormat("yyyy");
		String notice = 
		"/**\r\n"+
		" * Class : " + f.getName() +"\r\n"+
		" * Date  : " + sdf.format(new Date()) +"\r\n" + 
		" * Copyright Notice\r\n" +
		" *               Texas Education Agency\r\n"+                                       
		" ****************************************************************************\r\n" +
		" * THIS DOCUMENT CONTAINS MATERIAL WHICH IS THE PROPERTY OF AND CONFIDENTIAL \r\n" +
		" * TO THE TEXAS EDUCATION AGENCY. DISCLOSURE OUTSIDE THE TEXAS EDUCATION \r\n" +
		" * AGENCY IS PROHIBITED, EXCEPT BY LICENSE OR OTHER CONFIDENTIALITY \r\n" +
		" * AGREEMENT.\r\n" +
		" *\r\n"+
		" *   COPYRIGHT 2009-"+fY.format(new Date())+" THE TEXAS EDUCATION AGENCY. ALL RIGHTS RESERVED.\r\n"+
		" ****************************************************************************\r\n"+
		" */\r\n\r\n";
	
		out.write(notice);
	}
	
	
	public static void main(String args[]) throws Exception {
		UpdateHeaders uh = new UpdateHeaders();
		uh.update(System.getProperty("user.dir")+"/");
	}
}
