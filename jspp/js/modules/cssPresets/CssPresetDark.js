// @lx:module lx.CssPresetDark;

// @lx:namespace lx;
class CssPresetDark extends lx.CssPreset {
    static getName() {
        return 'dark';
    }

    getParams() {
        return {
            mainBackgroundColor: '#272822',
            altMainBackgroundColor: '#3E434C',
            bodyBackgroundColor: '#3C3F41',
            altBodyBackgroundColor: '#3C3B37',
            textBackgroundColor: '#45494A',
            widgetBackgroundColor: '#5B5E65',
            widgetGradient: 'linear-gradient(to bottom, #59574F, #403F3A)',
            widgetBorderColor: '#646464',
            widgetIconColor: '#DFD8B7',
            widgetColoredIconColor: '#272822',
            widgetIconColorDisabled: 'gray',
            headerTextColor: '#DFD8B7',
            textColor: '#BABABA',
            textColorDisabled: 'gray',
            shadowSize: 10,
            borderRadius: '5px',

            // Green
            checkedMainColor: '#378028',
            checkedSoftColor: '#50b73a',
            checkedLightColor: '#419d2f',
            checkedDarkColor: '#2f6a22',
            checkedDeepColor: '#255c19',
            // Blue
            coldMainColor: '#7553A6',
            coldSoftColor: '#9C84BE',
            coldLightColor: '#8568B0',
            coldDarkColor: '#6843A0',
            coldDeepColor: '#5F369C',
            // Red
            hotMainColor: '#ED6A76',
            hotSoftColor: '#F8A5AD',
            hotLightColor: '#F78892',
            hotDarkColor: '#E35260',
            hotDeepColor: '#DE3E4D',
            // Yellow
            neutralMainColor: '#F5E76E',
            neutralSoftColor: '#FFF6AA',
            neutralLightColor: '#FFF38C',
            neutralDarkColor: '#EBDB55',
            neutralDeepColor: '#E6D540'
        };
    }
}
