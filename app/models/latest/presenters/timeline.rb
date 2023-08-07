# typed: strict
# frozen_string_literal: true

class Latest::Presenters::Timeline < Latest::Presenters::Base
  sig { params(posts: ActiveRecord::Relation, page_info: PageInfo).void }
  def initialize(posts:, page_info:)
    @posts = posts
    @page_info = page_info
  end

  sig { params(args: T.untyped).returns(T::Hash[Symbol, T.untyped]) }
  def as_json(*args)
    {
      posts: Latest::Resources::Post.new(posts).to_h,
      page_info: Latest::Resources::PageInfo.new(page_info).to_h
    }
  end

  sig { returns(ActiveRecord::Relation) }
  attr_reader :posts
  private :posts

  sig { returns(PageInfo) }
  attr_reader :page_info
  private :page_info
end
