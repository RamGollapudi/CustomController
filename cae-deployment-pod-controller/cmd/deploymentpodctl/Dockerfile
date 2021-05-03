#FROM customimage

#RUN echo CACHEBUST>/dev/null && update-ca-certificates -f
    
ADD deploymentpodctl /bin/
RUN chmod +x /bin/deploymentpodctl
CMD ["/bin/deploymentpodctl"]
