# typed: strict
# frozen_string_literal: true

class Latest::Presenters::Timeline < Latest::Presenters::Base
  def initialize(posts:, page_info:)
    @posts = posts
    @page_info = page_info
  end

  def as_json(*)
    {
      posts: Resources::Latest::Post.new(posts).to_h,
      page_info: Resources::Latest::PageInfo.new(page_info).to_h
    }
  end

  attr_reader :posts
  private :posts

  attr_reader :page_info
  private :page_info
end
