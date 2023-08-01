# typed: strict
# frozen_string_literal: true

module Paginateable
  extend T::Sig
  extend ActiveSupport::Concern

  sig { params(page_info: PageInfo, path: String).void }
  def response_pagination_headers(page_info:, path:)
    link_items = []

    if page_info.has_previous_page
      link_items << %(<#{Rails.configuration.mewst["url"]}#{path}?before=#{page_info.start_cursor}>; rel="previous")
    end

    if page_info.has_next_page
      link_items << %(<#{Rails.configuration.mewst["url"]}#{path}?after=#{page_info.end_cursor}>; rel="next")
    end

    if link_items.present?
      response.set_header("Link", link_items.join(", "))
    end
  end
end
