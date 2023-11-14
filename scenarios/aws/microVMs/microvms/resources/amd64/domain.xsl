<?xml version="1.0"?>
<xsl:stylesheet version="1.0"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output omit-xml-declaration="yes" indent="yes"/>
  <xsl:template match="node()|@*">
      <xsl:copy>
         <xsl:apply-templates select="node()|@*"/>
      </xsl:copy>
   </xsl:template>
  <xsl:template match="/domain/features">
       <cpu mode='host-passthrough' check='none'>
           <model fallback='forbid'>qemu64</model>
       </cpu>
  </xsl:template>

  <xsl:template match="/domain/devices/disk">
      <filesystem type='mount' accessmode='passthrough'>
          <source dir='{sharedFSMount}'/>
          <target dir='kernel-version-testing'/>
      </filesystem>
      <readonly/>
      <xsl:copy>
          <xsl:apply-templates select="@*|node()"/>
      </xsl:copy>
  </xsl:template>

  <xsl:template match="/domain/devices/interface[@type='network']/mac/@address">
      <xsl:attribute name="address">
          <xsl:value-of select="'{mac}'"/>
      </xsl:attribute>
  </xsl:template>

  <xsl:template match="/domain/devices">
    <xsl:copy>
        <xsl:apply-templates select="node()|@*"/>
            <xsl:element name ="controller">
                <xsl:attribute name="type">usb</xsl:attribute>
            	<xsl:attribute name="model">
                    <xsl:value-of select="'none'"/>
            	</xsl:attribute>
            </xsl:element>
    </xsl:copy>
  </xsl:template>
  <xsl:template match="domain/devices/graphics"/>
  <xsl:template match="domain/devices/audio"/>
  <xsl:template match="domain/devices/video"/>
</xsl:stylesheet>

