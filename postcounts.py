#!/usr/bin/env python

import requests
import pytz
from pytz import timezone
from datetime import date, datetime, timedelta
import pprint

API_KEY = ## API Key goes here Get One from Admins
TARGET_URL = 'http://www.freedomplaybypost.com/api/'
DATE_FORMAT = '%Y-%m-%dT%XZ'
END_DATE = date.today().replace(day=1) - timedelta(days=1)
START_DATE = END_DATE.replace(day=1)
TARGET_FORUMS = '10,11,12,13,14,15,16,17,18,36,37,38,29,30,31,32,33,34,47,46,44,40,24,25,28,54,42,64,72,73,65,66,67,56,68,69,70,57,49,50'
TIME_ZONE = timezone('US/Eastern')
UTC = timezone('UTC')

def get_topics(url, rest_key):
    "Function Returns list of valid topic IDs and Titles from url authenticated by key"
    #valid_date = True
    end = False
    page = 0
    valid_topic_ids = []
    valid_topic_titles = {}

    while not end:
        page += 1
        topics = requests.get(url + 'forums/topics', auth=(rest_key, ''), params={'forums': TARGET_FORUMS,'page': page, 'sortBy': 'date', 'sortDir': 'desc', 'archived': 0, 'hidden': 0 }).json()
        results = topics.get('results')
        for topic in results:
            date_import = datetime.strptime(topic.get('lastPost').get('date'), DATE_FORMAT)
            date = UTC.localize(date_import).astimezone(TIME_ZONE).date()
            # print(date)
            # if date < START_DATE:
            #     end = True
            if date >= START_DATE:
                if topic.get('prefix') == 'ic':
                    valid_topic_ids.append(topic.get('id'))
                    valid_topic_titles[topic.get('id')] = (topic.get('title'), topic.get('url'))
        if int(topics.get('page')) >= topics.get('totalPages'):
            end = True
    return (valid_topic_ids, valid_topic_titles);

def get_titles_posts(url, rest_key):
    "return a list of dictionaries containing valid post details and a topic titles "
    valid_topics = get_topics(url=url, rest_key=rest_key)
    topic_ids = valid_topics[0]
    topic_titles = valid_topics[1]
    valid_posts = []
    for topic_id in topic_ids:
        last_page = False
        current_page = 0
        while not last_page:
            current_page += 1
            posts = requests.get(url + 'forums/topics/' + str(topic_id) + '/posts', auth=(rest_key, ''), params={'page': current_page, 'hidden': 0}).json()
            for post in posts.get('results'):
                post_date_import = datetime.strptime(post.get('date'), DATE_FORMAT)
                post_date = UTC.localize(post_date_import).astimezone(TIME_ZONE).date()
                if START_DATE <= post_date <= END_DATE:
                    validated_post = {'id':post.get('id'),
                                     'author_id': post.get('author').get('id'),
                                     'author_name': post.get('author').get('name').capitalize(),
                                     'parent_topic_id': post.get('item_id')}
                    valid_posts.append(validated_post)
            if current_page == posts.get('totalPages'):
                last_page = True
    return (valid_posts, topic_titles);

def get_user_counts(user, post_list):
    "return a user count dictionairy object with user_id, user_name, and user_thread_count keys; user_thread_count formated (topic_id, post count)"
    user_posts = []
    user_threads = []
    user_thread_counts = []
    for post in post_list:
        if post.get('parent_topic_id') not in user_threads:
            user_threads.append(post.get('parent_topic_id'))
        if post.get('author_id') == user[0]:
            user_posts.append(post.get('parent_topic_id'))
    for thread in user_threads:
        if user_posts.count(thread) > 0:
            user_thread_counts.append((thread, user_posts.count(thread)))
    user_count = {'user_id': user[0], 'user_name': user[1], 'user_thread_counts': user_thread_counts}
    return (user_count);

def get_user_list(post_list):
    "returns a list of author_id and author_name pairs from a list of posts"
    user_list = []
    for post in post_list:
        if (post.get('author_id'), post.get('author_name')) not in user_list:
            user_list.append((post.get('author_id'), post.get('author_name')))
    return (user_list);

def format_user_count(user_count):
    "Convert a user_count dictionairy into a formatted string for posting"
    threads_list = []
    for thread in user_count.get('user_thread_counts'):
        thread_string = '<li><a href="{url}" rel="">{thread_title}</a> = {count} Posts</li>'.format(url=titles_posts[1].get(thread[0])[1], thread_title=titles_posts[1].get(thread[0])[0], count=thread[1])
        threads_list.append(thread_string)
    threads_string = ''.join(threads_list)
    count_string = '<p><strong>{user_name}</strong></p><ul>{counts}</ul></p>'.format(user_name=user_count.get('user_name'), counts=threads_string)
    return (count_string);

print('Count Begining')

titles_posts = get_titles_posts(url=TARGET_URL, rest_key=API_KEY)
monthly_user_list = sorted(get_user_list(titles_posts[0]), key=lambda user: user[1])
user_count_list = []
final_post_list = []

# durf = get_user_counts(user=monthly_user_list[1], post_list=titles_posts[0]) # For testing monthly user list is sorted alphabetically

for user in monthly_user_list:
    user_count_list.append(get_user_counts(user=user, post_list=titles_posts[0]))
for count in user_count_list:
    final_post_list.append(format_user_count(user_count=count))


# pp = pprint.PrettyPrinter(indent=4)
# pp.pprint(durf)
# for post in titles_posts[0]:
#     if post.get('author_id') == 503:
#         print("ID: {} ThreadID: {}".format(post.get('id'), post.get('parent_topic_id')))
# pp.pprint(titles_posts[0])

final_post_string = ''.join(final_post_list)
final_post_title = '{} {} Post Counts'.format(START_DATE.strftime('%B'), START_DATE.year)
final_post_forum = 19
final_post_author = 3446340
final_post_params = {'forum': final_post_forum,
                     'author': final_post_author,
                     'title': final_post_title,
                     'post': final_post_string,
                     'prefix': 'Post Counts'}

# print(final_post_string)
response = requests.post(TARGET_URL + 'forums/topics', auth=(API_KEY, ''),  data=final_post_params)
print(response.json().get('title') + ' posted.')
print('Count Complete')
