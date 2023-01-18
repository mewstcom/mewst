# typed: strict
# frozen_string_literal: true

module ApplicationHelper
  extend T::Sig

  sig { returns(String) }
  def mewst_display_meta_tags
    display_meta_tags(
      reverse: true,
      site: "Mewst",
      separator: " |",
      description: "A microblogging service.",
      canonical: "#{request.protocol}#{request.host_with_port}#{request.path}"
    )
  end
end
