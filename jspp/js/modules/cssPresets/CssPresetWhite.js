// @lx:module lx.CssPresetWhite;

// @lx:namespace lx;
class CssPresetWhite extends lx.CssPreset {
    static getName() {
        return 'white';
    }

    getParams() {
        return {
            mainBackgroundColor: 'white',
            altMainBackgroundColor: '#F0F0F0',
            bodyBackgroundColor: 'white',
            altBodyBackgroundColor: '#F0F0F0',
            textBackgroundColor: 'white',
            widgetBackgroundColor: 'white',
            widgetGradient: 'linear-gradient(to bottom, #ECECEC, #D1D1D1)',
            widgetBorderColor: '#D9D9D9',
            widgetIconColor: 'black',
            widgetColoredIconColor: 'black',
            widgetIconColorDisabled: 'gray',
            headerTextColor: 'black',
            textColor: 'black',
            textColorDisabled: 'gray',
            shadowSize: 2,
            borderRadius: '5px',

            // Green
            checkedMainColor: '#6FCC5B',
            checkedSoftColor: '#A0DD93',
            checkedLightColor: '#86D676',
            checkedDarkColor: '#5DC447',
            checkedDeepColor: '#4DC035',
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
