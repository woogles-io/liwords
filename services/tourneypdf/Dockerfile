FROM public.ecr.aws/sam/build-python3.13

RUN dnf install -y cairo-devel pkgconfig google-noto-sans-fonts
WORKDIR /opt

# Copy the requirements.txt file into the Docker container
COPY requirements.txt .
RUN python3.13 -m pip install -r requirements.txt -t python/lib/python3.13/site-packages/

# Copy necessary libcairo shared libraries
RUN mkdir -p lib
RUN cp /usr/lib64/libcairo.so.2 lib/
RUN cp /usr/lib64/libpixman-1.so.0 lib/
# RUN cp /usr/lib64/libEGL.so.1 lib/
RUN cp /usr/lib64/libxcb-shm.so.0 lib/
RUN cp /usr/lib64/libxcb.so.1 lib/
RUN cp /usr/lib64/libxcb-render.so.0 lib/
RUN cp /usr/lib64/libXrender.so.1 lib/
RUN cp /usr/lib64/libX11.so.6 lib/
RUN cp /usr/lib64/libXext.so.6 lib/
# RUN cp /usr/lib64/libGL.so.1 lib/
# RUN cp /usr/lib64/libGLdispatch.so.0 lib/
# RUN cp /usr/lib64/libGLX.so.0 lib/
RUN cp /usr/lib64/libXau.so.6 lib/

# Copy fonts
RUN mkdir -p fonts
RUN cp -r /usr/share/fonts/google-noto fonts/
COPY fonts.conf /opt/fonts

# Package everything into a zip file
CMD zip -r /output/pycairo-layer.zip python lib fonts
