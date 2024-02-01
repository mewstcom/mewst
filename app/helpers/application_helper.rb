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
      description: t("meta.description.default"),
      canonical: "#{request.protocol}#{request.host_with_port}#{request.path}",
      og: {
        title: meta_tags.full_title(site: "Mewst", separator: " |"),
        type: "website",
        url: request.url,
        description: t("meta.description.default"),
        site_name: "Mewst",
        image: "#{request.protocol}#{request.host_with_port}/og-image.png",
        locale: (I18n.locale == :ja) ? "ja_JP" : "en_US"
      }
    )
  end
end
