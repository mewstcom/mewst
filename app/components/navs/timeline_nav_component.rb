# typed: strict
# frozen_string_literal: true

class Navs::TimelineNavComponent < ApplicationComponent
  sig { params(path: String).returns(String) }
  def nav_link_class_names(path)
    class_names("nav-link", {active: current_page?(path)})
  end
end
