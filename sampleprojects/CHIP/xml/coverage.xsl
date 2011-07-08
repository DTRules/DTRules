<?xml version="1.0" encoding="ISO-8859-1"?>

<xsl:stylesheet version="1.0"
xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<xsl:template match="/">
  <html>
  <body>
  <h2>Coverage</h2>
  <table border="1">
    <tr bgcolor="#9acd32">
      <th>Table</th>
      <th>column</th>
    </tr>
    <xsl:for-each select="coverage/table">
    <tr>
      <td>1</td>
      <td><xsl:value-of select="@name"/></td>
      <td><xsl:value-of select="@count_of_calls"/></td>
    </tr>
    </xsl:for-each>
  </table>
  </body>
  </html>
</xsl:template>

</xsl:stylesheet>
