# typed: strict
# frozen_string_literal: true

class Icons::LogoIconComponent < ApplicationComponent
  erb_template <<~ERB
    <svg class="c-logo-icon u-svg-icon" height="32" width="24" viewBox="0 0 24 32">
      <path d="M7.63924 24.6077L5.11582 22.0386C4.58062 21.4939 4.19031 20.5338 4.19031 19.7627V15.0624C4.19031 14.7751 3.99714 14.2661 3.80736 14.0535L0.122438 9.92683C-0.0537432 9.72955 -0.0367577 9.43238 0.153848 9.25594L0.831454 8.62877C1.02503 8.44962 1.31713 8.4666 1.49044 8.66065L5.17536 12.7873C5.67462 13.3464 6.03908 14.3068 6.03908 15.0623V19.7626C6.03908 20.0346 6.23481 20.5158 6.42316 20.7075L8.23838 22.5556L10.7937 14.5342C12.1045 15.7384 13.8408 16.4714 15.7455 16.4714C17.6502 16.4714 19.3864 15.7384 20.6973 14.534L23.613 23.6869C25.0523 28.205 22.3448 32 17.6824 32H13.8084C9.4663 32 6.82041 28.7082 7.63916 24.6076L7.63924 24.6077ZM20.1196 0.340493C20.6731 -0.335709 21.7541 0.0628017 21.7541 0.942966V8.93958C21.7541 12.3197 19.0645 15.0598 15.7455 15.0598C12.4265 15.0598 9.73683 12.3202 9.73683 8.93958V0.942966C9.73683 0.0628017 10.8178 -0.335709 11.3714 0.340493L13.4052 2.82526H18.0857L20.1196 0.340493Z"></path>
    </svg>
  ERB
end