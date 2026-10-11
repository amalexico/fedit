#!/usr/bin/env python
"""
Title: Disk space information provider
About: This program is used to measure the space in isilon and send an email with it
Requested by: Carlos Wood
Created by: Alejandro Castaneda
Date: 1/22/2019
Version: 1.00

"""
import smtplib, commands


class DiskInfo1:
    """
    This class defines all the functions for Disk information retrieval and forwarding
    """
    
    def __init__(self):
        """
        Class initialization 
        """
        # Path were the subtitles are located
        self.fromaddr = 'nearline01@hbo-la.com'
        self.toaddrs = 'cwood@hbo-la.com,rlutchman@hbo-la.com,acastaneda@hbo-la.com'
        self.ResultLine = ''
        
            
    def calculateValues1(self):
        """
        Obtain partition information
        """
        OutputResult = commands.getoutput("df -h | grep OneFS")
        #print("{0}".format(OutputResult))
        values = OutputResult.split()
        #print values
        self.ResultLine = "Nearline01 is using {0} of the total capacity of {1} = {2}.".format(values[2], values[1], values[4])
        #print self.ResultLine

    def sendEmail1(self):
        """
        Send the values by email
        """
        try:
            server = smtplib.SMTP('172.18.2.129')
            server.set_debuglevel(1)
            msg = "Subject: {0}\n\n{1}".format("Report of Nearline01 space",self.ResultLine)
            server.sendmail(self.fromaddr, self.toaddrs.split(','), msg)
            server.quit()
            return True
        except:
            return False

    def mainProgram1(self):
        """
        Executes the program
        """
        self.calculateValues()
        if self.sendEmail():
            print ("Successful")
        else:
            print("Failed to send email")

if __name__ == "__main__":
    Di = DiskInfo1()
    Di.mainProgram1()
    
    
    