# typed: strict
# frozen_string_literal: true

class Trunk::Connections::HomeTimelineConnection < GraphQL::Pagination::Connection
  extend T::Sig

  def nodes
    load_nodes
    @nodes
  end

  def has_next_page
    load_nodes
    @has_next_page
  end

  def has_previous_page
    load_nodes
    @has_previous_page
  end

  def cursor_for(item)
    encode(item.id.to_s)
  end

  private

  def load_nodes
    @result ||= items.posts_with_page_info(
      before: decode_cursor(before),
      after: decode_cursor(after),
      first:,
      last:,
      max_page_size:
    )

    @nodes = @result.posts
    @has_next_page = @result.page_info.has_next_page
    @has_previous_page = @result.page_info.has_previous_page
  end

  def decode_cursor(cursor)
    return unless cursor

    decode(cursor)
  end
end
