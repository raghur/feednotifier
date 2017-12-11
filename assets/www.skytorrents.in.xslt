<?xml version="1.0"?>
<xsl:stylesheet version="1.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
 xmlns:atom="http://www.w3.org/2005/Atom"   xmlns:set="http://exslt.org/sets">
    <xsl:param name="originalfile"/>
    <!-- <xsl:template match="text()|@*"> -->
    <!-- </xsl:template> -->
    <xsl:output method="xml" indent="yes"/>
    <xsl:template match="node()|text()">
        <xsl:copy>
            <xsl:apply-templates/>
        </xsl:copy>
    </xsl:template>
    <xsl:template match="/rss/channel/item">
        <xsl:variable name="val" select="guid"/>
        <!-- Second template match: Check if guid <xsl:value-of select="$val"/> exists -->
        <xsl:choose>
            <xsl:when test="document($originalfile)//guid[text()=$val]">
            </xsl:when>
            <xsl:otherwise>
                <xsl:copy-of select="."/>
            </xsl:otherwise>
        </xsl:choose>
    </xsl:template>
</xsl:stylesheet>
