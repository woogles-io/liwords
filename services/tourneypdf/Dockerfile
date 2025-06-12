FROM public.ecr.aws/sam/build-python3.13

RUN dnf install -y cairo-devel pkgconfig google-noto-sans-fonts
WORKDIR /opt

# Copy the requirements.txt file into the Docker container
COPY requirements.txt .
RUN python3.13 -m pip install --upgrade pip
RUN python3.13 -m pip install wheel setuptools
RUN PYTHONPATH=/opt/python python3.13 -m pip install -r requirements.txt --target /opt/python --no-cache-dir

# Copy necessary libcairo shared libraries
RUN echo "Copying all required libraries..." && \
    mkdir -p lib && \
    for LIB in libcairo.so.2 libpixman-1.so.0 libxcb-shm.so.0 libxcb.so.1 \
    libxcb-render.so.0 libXrender.so.1 libX11.so.6 libXext.so.6 \
    libXau.so.6 libpng16.so.16 libfontconfig.so.1 libfreetype.so.6 \
    libxml2.so.2 libharfbuzz.so.0 libbrotlidec.so.1 libbrotlicommon.so.1 \
    libglib-2.0.so.0 libgraphite2.so.3 libpcre2-8.so.0; do \
    cp -v /usr/lib64/$LIB lib/ || echo "Warning: Could not find $LIB"; \
    done

RUN ldd /opt/python/cairo/_cairo.*.so | grep "not found" || echo "All dependencies found"


# Copy fonts
RUN mkdir -p fonts
RUN cp -r /usr/share/fonts/google-noto fonts/
COPY fonts.conf /opt/fonts

# Package everything into a zip file
CMD zip -r /output/pycairo-layer.zip python lib fonts
